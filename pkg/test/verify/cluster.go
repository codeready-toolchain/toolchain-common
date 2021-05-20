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
			toolchainCluster, sec := test.NewToolchainCluster("east", "secret", status, labels)
			cl := test.NewFakeClient(t, toolchainCluster, sec)
			service := newToolchainClusterService(cl)
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
			toolchainCluster, sec := test.NewToolchainCluster("east", "secret", status, labels)
			cl := test.NewFakeClient(t, toolchainCluster, sec)
			service := newToolchainClusterService(cl)
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
		})
	}
}

func AddToolchainClusterFailsBecauseOfMissingSecret(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionTrue)
	toolchainCluster, _ := test.NewToolchainCluster("east", "secret", status, Labels("", "", test.NameHost))
	cl := test.NewFakeClient(t, toolchainCluster)
	service := newToolchainClusterService(cl)

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
	service := newToolchainClusterService(cl)

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
	service := newToolchainClusterService(cl)
	defer service.DeleteToolchainCluster("east")
	err := service.AddOrUpdateToolchainCluster(toolchainCluster1)
	require.NoError(t, err)

	// when
	err = functionToVerify(toolchainCluster2, cl, service)

	// then
	require.NoError(t, err)
	cachedToolchainCluster, ok := cluster.GetCachedToolchainCluster("east")
	require.True(t, ok)
	assert.Equal(t, cluster.Host, cachedToolchainCluster.Type)
	assert.Equal(t, "toolchain-host-operator", cachedToolchainCluster.OperatorNamespace)
	assert.Equal(t, statusFalse, *cachedToolchainCluster.ClusterStatus)
	assert.Equal(t, test.NameMember, cachedToolchainCluster.OwnerClusterName)
	assert.Equal(t, "http://cluster.com", cachedToolchainCluster.APIEndpoint)
}

func DeleteToolchainCluster(t *testing.T, functionToVerify FunctionToVerify) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionTrue)
	toolchainCluster, sec := test.NewToolchainCluster("east", "sec", status,
		Labels("", "", test.NameHost))
	cl := test.NewFakeClient(t, sec)
	service := newToolchainClusterService(cl)
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
	}
	if ns != "" {
		labels["namespace"] = ns
	}
	labels["ownerClusterName"] = ownerClusterName
	return labels
}

func newToolchainClusterService(cl client.Client) cluster.ToolchainClusterService {
	return cluster.NewToolchainClusterService(cl, logf.Log, "test-namespace", 3*time.Second)
}
