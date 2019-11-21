package cluster

import (
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"testing"
)

func TestRefreshCacheInService(t *testing.T) {
	// given
	defer gock.Off()
	status := test.NewClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	kubeFedCluster, sec := test.NewKubeFedCluster("east", "secret", status, map[string]string{"ownerClusterName": test.NameMember})
	cl := test.NewFakeClient(t, kubeFedCluster, sec)
	service := NewKubeFedClusterService(cl, logf.Log, "test-namespace")

	t.Run("the member cluster should be retrieved when refreshCache func is called", func(t *testing.T) {
		// given
		_, ok := clusterCache.clusters["east"]
		require.False(t, ok)
		defer service.DeleteKubeFedCluster(kubeFedCluster)

		// when
		service.refreshCache()

		// then
		fedCluster, ok := GetFedCluster(test.NameMember)
		require.True(t, ok)
		assertMemberCluster(t, fedCluster, status)
	})

	t.Run("the member cluster should be retrieved when GetFedCluster func is called", func(t *testing.T) {
		// given
		_, ok := clusterCache.clusters["east"]
		require.False(t, ok)
		defer service.DeleteKubeFedCluster(kubeFedCluster)

		// when
		fedCluster, ok := GetFedCluster(test.NameMember)

		// then
		require.True(t, ok)
		assertMemberCluster(t, fedCluster, status)
	})

	t.Run("the host cluster should not be retrieved", func(t *testing.T) {
		// given
		_, ok := clusterCache.clusters["east"]
		require.False(t, ok)
		defer service.DeleteKubeFedCluster(kubeFedCluster)

		// when
		fedCluster, ok := GetFedCluster(test.NameHost)

		// then
		require.False(t, ok)
		assert.Nil(t, fedCluster)
	})
}

func assertMemberCluster(t *testing.T, fedCluster *FedCluster, status v1beta1.KubeFedClusterStatus) {
	assert.Equal(t, Member, fedCluster.Type)
	assert.Equal(t, status, *fedCluster.ClusterStatus)
	assert.Equal(t, test.NameMember, fedCluster.OwnerClusterName)
	assert.Equal(t, "http://cluster.com", fedCluster.APIEndpoint)
}
