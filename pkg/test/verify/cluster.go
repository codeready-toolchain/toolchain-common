package verify

import (
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
	status := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionTrue)
	memberLabels := []map[string]string{
		Labels("", "", test.NameHost),
		Labels(cluster.Member, "", test.NameHost),
		Labels(cluster.Member, "member-ns", test.NameHost)}
	for _, labels := range memberLabels {

		t.Run("add member ToolchainCluster", func(t *testing.T) {
			for _, withCA := range []bool{true, false} {
				toolchainCluster, sec := test.NewToolchainCluster("east", "secret", status, labels)
				if withCA {
					toolchainCluster.Spec.CABundle = "ZHVtbXk="
				}
				cl := test.NewFakeClient(t, toolchainCluster, sec)
				service := newToolchainClusterService(t, cl, withCA)
				defer service.DeleteToolchainCluster("east")

				// when
				err := functionToVerify(toolchainCluster, cl, service)

				// then
				require.NoError(t, err)
				cachedToolchainCluster, ok := cluster.GetCachedToolchainCluster("east")
				require.True(t, ok)
				assert.Equal(t, cluster.Member, cachedToolchainCluster.Type)
				if labels["namespace"] == "" {
					assert.Equal(t, "toolchain-member-operator", cachedToolchainCluster.OperatorNamespace)
				} else {
					assert.Equal(t, labels["namespace"], cachedToolchainCluster.OperatorNamespace)
				}
				assert.Equal(t, status, *cachedToolchainCluster.ClusterStatus)
				assert.Equal(t, test.NameHost, cachedToolchainCluster.OwnerClusterName)
				assert.Equal(t, "http://cluster.com", cachedToolchainCluster.APIEndpoint)
			}
		})
	}
}

func AddToolchainClusterAsHost(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionFalse)
	memberLabels := []map[string]string{
		Labels(cluster.Host, "", test.NameMember),
		Labels(cluster.Host, "host-ns", test.NameMember)}
	for _, labels := range memberLabels {

		t.Run("add host ToolchainCluster", func(t *testing.T) {
			for _, withCA := range []bool{true, false} {
				toolchainCluster, sec := test.NewToolchainCluster("east", "secret", status, labels)
				if withCA {
					toolchainCluster.Spec.CABundle = "ZHVtbXk="
				}
				cl := test.NewFakeClient(t, toolchainCluster, sec)
				service := newToolchainClusterService(t, cl, withCA)
				defer service.DeleteToolchainCluster("east")

				// when
				err := functionToVerify(toolchainCluster, cl, service)

				// then
				require.NoError(t, err)
				cachedToolchainCluster, ok := cluster.GetCachedToolchainCluster("east")
				require.True(t, ok)
				assert.Equal(t, cluster.Host, cachedToolchainCluster.Type)
				if labels["namespace"] == "" {
					assert.Equal(t, "toolchain-host-operator", cachedToolchainCluster.OperatorNamespace)
				} else {
					assert.Equal(t, labels["namespace"], cachedToolchainCluster.OperatorNamespace)
				}
				assert.Equal(t, status, *cachedToolchainCluster.ClusterStatus)
				assert.Equal(t, test.NameMember, cachedToolchainCluster.OwnerClusterName)
				assert.Equal(t, "http://cluster.com", cachedToolchainCluster.APIEndpoint)
			}
		})
	}
}

func AddToolchainClusterFailsBecauseOfMissingSecret(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionTrue)
	toolchainCluster, _ := test.NewToolchainCluster("east", "secret", status, Labels("", "", test.NameHost))
	cl := test.NewFakeClient(t, toolchainCluster)
	service := newToolchainClusterService(t, cl, false)

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
	status := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionTrue)
	toolchainCluster, _ := test.NewToolchainCluster("east", "secret", status,
		Labels("", "", test.NameHost))
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "secret",
			Namespace: "test-namespace",
		}}
	cl := test.NewFakeClient(t, toolchainCluster, secret)
	service := newToolchainClusterService(t, cl, false)

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
	statusTrue := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionTrue)
	toolchainCluster1, sec1 := test.NewToolchainCluster("east", "secret1", statusTrue,
		Labels("", "", test.NameMember))
	statusFalse := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionFalse)
	toolchainCluster2, sec2 := test.NewToolchainCluster("east", "secret2", statusFalse,
		Labels(cluster.Host, "", test.NameMember))
	cl := test.NewFakeClient(t, toolchainCluster2, sec1, sec2)
	service := newToolchainClusterService(t, cl, false)
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
		IsOfType(cluster.Host).
		HasName("east").
		HasOperatorNamespace("toolchain-host-operator").
		HasOwnerClusterName(test.NameMember).
		HasAPIEndpoint("http://cluster.com").
		RestConfigHasHost("http://cluster.com")
}

func DeleteToolchainCluster(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionTrue)
	toolchainCluster, sec := test.NewToolchainCluster("east", "sec", status,
		Labels("", "", test.NameHost))
	cl := test.NewFakeClient(t, sec)
	service := newToolchainClusterService(t, cl, false)
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

func Labels(clType cluster.Type, ns, ownerClusterName string) map[string]string {
	labels := map[string]string{}
	if clType != "" {
		labels["type"] = string(clType)
		labels["toolchain.dev.openshift.com/cluster-label/"+string(clType)] = ""
	}
	if ns != "" {
		labels["namespace"] = ns
	}
	labels["ownerClusterName"] = ownerClusterName
	return labels
}

func newToolchainClusterService(t *testing.T, cl client.Client, withCA bool) cluster.ToolchainClusterService {
	return cluster.NewToolchainClusterServiceWithClient(cl, logf.Log, "test-namespace", 3*time.Second, func(config *rest.Config, options client.Options) (client.Client, error) {
		if withCA {
			assert.False(t, config.Insecure)
			assert.Equal(t, []byte("dummy"), config.CAData)
		} else {
			assert.True(t, config.Insecure)
		}
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

func (a *ClusterConfigAssertion) IsOfType(clusterType cluster.Type) *ClusterConfigAssertion {
	assert.Equal(a.t, clusterType, a.clusterConfig.Type)
	return a
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

func (a *ClusterConfigAssertion) RestConfigHasHost(host string) *ClusterConfigAssertion {
	require.NotNil(a.t, a.clusterConfig.RestConfig)
	assert.Equal(a.t, host, a.clusterConfig.RestConfig.Host)
	return a
}
