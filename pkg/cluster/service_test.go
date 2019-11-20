package cluster_test

import (
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"testing"
)

func TestAddKubeFedClusterAsMember(t *testing.T) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	memberLabels := []map[string]string{
		labels("", "", test.NameHost),
		labels(cluster.Member, "", test.NameHost),
		labels(cluster.Member, "member-ns", test.NameHost)}
	for _, labels := range memberLabels {

		t.Run("add member KubeFedCluster", func(t *testing.T) {
			kubeFedCluster, sec := test.NewKubeFedCluster("east", "secret", status, labels)
			cl := test.NewFakeClient(t, sec)
			service := cluster.NewKubeFedClusterService(cl, logf.Log, "test-namespace")
			defer service.DeleteKubeFedCluster(kubeFedCluster)

			// when
			service.AddKubeFedCluster(kubeFedCluster)

			// then
			fedCluster, ok := cluster.GetFedCluster("east")
			require.True(t, ok)
			assert.Equal(t, cluster.Member, fedCluster.Type)
			if labels["namespace"] == "" {
				assert.Equal(t, "toolchain-member-operator", fedCluster.OperatorNamespace)
			} else {
				assert.Equal(t, labels["namespace"], fedCluster.OperatorNamespace)
			}
			assert.Equal(t, status, *fedCluster.ClusterStatus)
			assert.Equal(t, test.NameHost, fedCluster.OwnerClusterName)
			assert.Equal(t, "http://cluster.com", fedCluster.APIEndpoint)
		})
	}
}

func TestAddKubeFedClusterAsHost(t *testing.T) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(common.ClusterReady, corev1.ConditionFalse)
	memberLabels := []map[string]string{
		labels(cluster.Host, "", test.NameMember),
		labels(cluster.Host, "host-ns", test.NameMember)}
	for _, labels := range memberLabels {

		t.Run("add host KubeFedCluster", func(t *testing.T) {
			kubeFedCluster, sec := test.NewKubeFedCluster("east", "secret", status, labels)
			cl := test.NewFakeClient(t, sec)
			service := cluster.NewKubeFedClusterService(cl, logf.Log, "test-namespace")
			defer service.DeleteKubeFedCluster(kubeFedCluster)

			// when
			service.AddKubeFedCluster(kubeFedCluster)

			// then
			fedCluster, ok := cluster.GetFedCluster("east")
			require.True(t, ok)
			assert.Equal(t, cluster.Host, fedCluster.Type)
			if labels["namespace"] == "" {
				assert.Equal(t, "toolchain-host-operator", fedCluster.OperatorNamespace)
			} else {
				assert.Equal(t, labels["namespace"], fedCluster.OperatorNamespace)
			}
			assert.Equal(t, status, *fedCluster.ClusterStatus)
			assert.Equal(t, test.NameMember, fedCluster.OwnerClusterName)
			assert.Equal(t, "http://cluster.com", fedCluster.APIEndpoint)
		})
	}
}

func TestAddKubeFedClusterFailsBecauseOfMissingSecret(t *testing.T) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	kubeFedCluster, _ := test.NewKubeFedCluster("east", "secret", status, labels("", "", test.NameHost))
	cl := test.NewFakeClient(t, )
	service := cluster.NewKubeFedClusterService(cl, logf.Log, "test-namespace")

	// when
	service.AddKubeFedCluster(kubeFedCluster)

	// then
	fedCluster, ok := cluster.GetFedCluster("east")
	require.False(t, ok)
	assert.Nil(t, fedCluster)
}

func TestAddKubeFedClusterFailsBecauseOfEmptySecret(t *testing.T) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	kubeFedCluster, _ := test.NewKubeFedCluster("east", "secret", status,
		labels("", "", test.NameHost))
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "secret",
			Namespace: "test-namespace",
		}}
	cl := test.NewFakeClient(t, secret)
	service := cluster.NewKubeFedClusterService(cl, logf.Log, "test-namespace")

	// when
	service.AddKubeFedCluster(kubeFedCluster)

	// then
	fedCluster, ok := cluster.GetFedCluster("east")
	require.False(t, ok)
	assert.Nil(t, fedCluster)
}

func TestUpdateKubeFedCluster(t *testing.T) {
	// given
	defer gock.Off()
	statusTrue := test.NewClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	kubeFedCluster1, sec1 := test.NewKubeFedCluster("east", "secret1", statusTrue,
		labels("", "", test.NameMember))
	statusFalse := test.NewClusterStatus(common.ClusterReady, corev1.ConditionFalse)
	kubeFedCluster2, sec2 := test.NewKubeFedCluster("east", "secret2", statusFalse,
		labels(cluster.Host, "", test.NameMember))
	cl := test.NewFakeClient(t, sec1, sec2)
	service := cluster.NewKubeFedClusterService(cl, logf.Log, "test-namespace")
	defer service.DeleteKubeFedCluster(kubeFedCluster2)
	service.AddKubeFedCluster(kubeFedCluster1)

	// when
	service.AddKubeFedCluster(kubeFedCluster2)

	// then
	fedCluster, ok := cluster.GetFedCluster("east")
	require.True(t, ok)
	assert.Equal(t, cluster.Host, fedCluster.Type)
	assert.Equal(t, "toolchain-host-operator", fedCluster.OperatorNamespace)
	assert.Equal(t, statusFalse, *fedCluster.ClusterStatus)
	assert.Equal(t, test.NameMember, fedCluster.OwnerClusterName)
	assert.Equal(t, "http://cluster.com", fedCluster.APIEndpoint)
}

func TestDeleteKubeFedCluster(t *testing.T) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	kubeFedCluster, sec := test.NewKubeFedCluster("east", "sec", status,
		labels("", "", test.NameHost))
	cl := test.NewFakeClient(t, sec)
	service := cluster.NewKubeFedClusterService(cl, logf.Log, "test-namespace")
	service.AddKubeFedCluster(kubeFedCluster)

	// when
	service.DeleteKubeFedCluster(kubeFedCluster)

	// then
	fedCluster, ok := cluster.GetFedCluster("east")
	require.False(t, ok)
	assert.Nil(t, fedCluster)
}

func labels(clType cluster.Type, ns, ownerClusterName string) map[string]string {
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
