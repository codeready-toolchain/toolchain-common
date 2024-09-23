package verify

import (
	"context"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type FunctionToVerify func(toolchainCluster *toolchainv1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error

func AddToolchainClusterAsMember(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(toolchainv1alpha1.ConditionReady, corev1.ConditionTrue)

	toolchainCluster, sec := test.NewToolchainCluster(t, "east", test.HostOperatorNs, "member-ns", "secret", status, false)

	cl := test.NewFakeClient(t, toolchainCluster, sec)
	service := newToolchainClusterService(t, cl, false)
	defer service.DeleteToolchainCluster("east")
	// when
	err := functionToVerify(toolchainCluster, cl, service)
	// then
	require.NoError(t, err)
	cachedToolchainCluster, ok := cluster.GetCachedToolchainCluster("east")
	require.True(t, ok)
	assert.Equal(t, "member-ns", cachedToolchainCluster.OperatorNamespace)
	require.NoError(t, cl.Get(context.TODO(), client.ObjectKeyFromObject(toolchainCluster), toolchainCluster))
	assert.Equal(t, status, *cachedToolchainCluster.ClusterStatus)
	assert.Equal(t, "https://cluster.com", cachedToolchainCluster.APIEndpoint)
}

func AddToolchainClusterAsHost(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(toolchainv1alpha1.ConditionReady, corev1.ConditionFalse)

	toolchainCluster, sec := test.NewToolchainCluster(t, "east", test.MemberOperatorNs, "host-ns", "secret", status, false)
	cl := test.NewFakeClient(t, toolchainCluster, sec)
	service := newToolchainClusterService(t, cl, false)
	defer service.DeleteToolchainCluster("east")

	// when
	err := functionToVerify(toolchainCluster, cl, service)

	// then
	require.NoError(t, err)
	cachedToolchainCluster, ok := cluster.GetCachedToolchainCluster("east")
	require.True(t, ok)

	assert.Equal(t, "host-ns", cachedToolchainCluster.OperatorNamespace)

	require.NoError(t, cl.Get(context.TODO(), client.ObjectKeyFromObject(toolchainCluster), toolchainCluster))
	assert.Equal(t, status, *cachedToolchainCluster.ClusterStatus)
	assert.Equal(t, "https://cluster.com", cachedToolchainCluster.APIEndpoint)
}

func AddToolchainClusterFailsBecauseOfMissingSecret(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(toolchainv1alpha1.ConditionReady, corev1.ConditionTrue)
	toolchainCluster, _ := test.NewToolchainCluster(t, "east", test.MemberOperatorNs, "", "secret", status, false)
	cl := test.NewFakeClient(t, toolchainCluster)
	service := newToolchainClusterService(t, cl, true)

	// when
	err := functionToVerify(toolchainCluster, cl, service)

	// then
	require.Error(t, err)
	cachedToolchainCluster, ok := cluster.GetCachedToolchainCluster("east")
	require.False(t, ok)
	assert.Nil(t, cachedToolchainCluster)
}

func AddToolchainClusterFailsBecauseOfEmptySecret(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(toolchainv1alpha1.ConditionReady, corev1.ConditionTrue)
	toolchainCluster, _ := test.NewToolchainCluster(t, "east", test.MemberOperatorNs, test.MemberOperatorNs, "secret", status, false)
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "secret",
			Namespace: test.MemberOperatorNs,
		},
	}
	cl := test.NewFakeClient(t, toolchainCluster, secret)
	service := newToolchainClusterService(t, cl, true)

	// when
	err := functionToVerify(toolchainCluster, cl, service)

	// then
	require.Error(t, err)
	cachedToolchainCluster, ok := cluster.GetCachedToolchainCluster("east")
	require.False(t, ok)
	assert.Nil(t, cachedToolchainCluster)
}

func UpdateToolchainCluster(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	statusTrue := test.NewClusterStatus(toolchainv1alpha1.ConditionReady, corev1.ConditionTrue)
	toolchainCluster1, sec1 := test.NewToolchainCluster(t, "east", test.HostOperatorNs, test.HostOperatorNs, "secret1", statusTrue, true)
	statusFalse := test.NewClusterStatus(toolchainv1alpha1.ConditionReady, corev1.ConditionFalse)
	toolchainCluster2, sec2 := test.NewToolchainCluster(t, "east", test.HostOperatorNs, test.HostOperatorNs, "secret2", statusFalse, true)
	cl := test.NewFakeClient(t, toolchainCluster2, sec1, sec2)
	service := newToolchainClusterService(t, cl, true)
	defer service.DeleteToolchainCluster("east")
	err := service.AddOrUpdateToolchainCluster(toolchainCluster1)
	require.NoError(t, err)

	// when
	err = functionToVerify(toolchainCluster2, cl, service)

	// then
	require.NoError(t, err)
	cachedToolchainCluster, ok := cluster.GetCachedToolchainCluster("east")
	require.True(t, ok)
	assert.Equal(t, statusFalse, *cachedToolchainCluster.ClusterStatus)
	AssertClusterConfigThat(t, cachedToolchainCluster.Config).
		HasName("east").
		HasOperatorNamespace("toolchain-host-operator").
		HasAPIEndpoint("https://cluster.com").
		RestConfigHasHost("https://cluster.com")
}

func DeleteToolchainCluster(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(toolchainv1alpha1.ConditionReady, corev1.ConditionTrue)
	toolchainCluster, sec := test.NewToolchainCluster(t, "east", test.MemberOperatorNs, test.MemberOperatorNs, "sec", status, true)
	cl := test.NewFakeClient(t, sec)
	service := newToolchainClusterService(t, cl, true)
	err := service.AddOrUpdateToolchainCluster(toolchainCluster)
	require.NoError(t, err)

	// when
	err = functionToVerify(toolchainCluster, cl, service)

	// then
	require.NoError(t, err)
	cachedToolchainCluster, ok := cluster.GetCachedToolchainCluster("east")
	require.False(t, ok)
	assert.Nil(t, cachedToolchainCluster)
}

func newToolchainClusterService(t *testing.T, cl client.Client, insecure bool) cluster.ToolchainClusterService {
	return cluster.NewToolchainClusterServiceWithClient(cl, logf.Log, "test-namespace", 3*time.Second, func(config *rest.Config, options client.Options) (client.Client, error) {
		if insecure {
			assert.True(t, config.Insecure)
		} else {
			assert.False(t, config.Insecure)
		}
		assert.Empty(t, config.CAData)
		// make sure that insecure is false to make Gock mocking working properly
		config.Insecure = false
		// reset the dummy certificate
		config.CAData = []byte("")
		return client.New(config, options)
	})
}

type ClusterConfigAssertion struct {
	t             *testing.T
	clusterConfig *cluster.Config
}

func AssertClusterConfigThat(t *testing.T, clusterConfig *cluster.Config) *ClusterConfigAssertion {
	return &ClusterConfigAssertion{
		t:             t,
		clusterConfig: clusterConfig,
	}
}

func (a *ClusterConfigAssertion) HasOperatorNamespace(namespace string) *ClusterConfigAssertion {
	assert.Equal(a.t, namespace, a.clusterConfig.OperatorNamespace)
	return a
}

func (a *ClusterConfigAssertion) HasName(name string) *ClusterConfigAssertion {
	assert.Equal(a.t, name, a.clusterConfig.Name)
	return a
}

func (a *ClusterConfigAssertion) HasOwnerClusterName(name string) *ClusterConfigAssertion {
	assert.Equal(a.t, name, a.clusterConfig.OwnerClusterName)
	return a
}

func (a *ClusterConfigAssertion) HasAPIEndpoint(apiEndpoint string) *ClusterConfigAssertion {
	assert.Equal(a.t, apiEndpoint, a.clusterConfig.APIEndpoint)
	return a
}

func (a *ClusterConfigAssertion) ContainsLabel(label string) *ClusterConfigAssertion {
	assert.Contains(a.t, a.clusterConfig.Labels, label)
	return a
}

func (a *ClusterConfigAssertion) RestConfigHasHost(host string) *ClusterConfigAssertion {
	require.NotNil(a.t, a.clusterConfig.RestConfig)
	assert.Equal(a.t, host, a.clusterConfig.RestConfig.Host)
	return a
}
