package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

var getFedClusterFuncs = []func(name string) (*FedCluster, bool){
	clusterCache.getFedCluster, GetFedCluster}

func TestAddCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	fedCluster := newTestFedCluster("testCluster", Member, v1.ConditionTrue)

	// when
	clusterCache.addFedCluster(fedCluster)

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, fedCluster, clusterCache.clusters["testCluster"])
}

func TestGetCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	fedCluster := newTestFedCluster("testCluster", Member, v1.ConditionTrue)
	clusterCache.addFedCluster(fedCluster)
	clusterCache.addFedCluster(newTestFedCluster("cluster", Member, v1.ConditionTrue))

	for _, getFedCluster := range getFedClusterFuncs {

		// when
		returnedFedCluster, ok := getFedCluster("testCluster")

		// then
		assert.True(t, ok)
		assert.Equal(t, fedCluster, returnedFedCluster)
	}
}

func TestHostCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	host := newTestFedCluster("host-cluster", Host, v1.ConditionTrue)
	clusterCache.addFedCluster(host)

	// when
	returnedFedCluster, ok := HostCluster()

	// then
	assert.True(t, ok)
	assert.Equal(t, host, returnedFedCluster)
}

func TestMemberClusters(t *testing.T) {
	// given
	defer resetClusterCache()
	member1 := newTestFedCluster("member-cluster-1", Member, v1.ConditionTrue)
	clusterCache.addFedCluster(member1)
	member2 := newTestFedCluster("member-cluster-2", Member, v1.ConditionTrue)
	clusterCache.addFedCluster(member2)

	// when
	returnedFedClusters := MemberClusters()

	// then
	require.Len(t, returnedFedClusters, 2)
	assert.Equal(t, returnedFedClusters[0], member1)
	assert.Equal(t, returnedFedClusters[1], member2)
}

func TestGetClusterWhenIsEmpty(t *testing.T) {
	// given
	resetClusterCache()

	for _, getFedCluster := range getFedClusterFuncs {

		// when
		returnedFedCluster, ok := getFedCluster("testCluster")

		// then
		assert.False(t, ok)
		assert.Nil(t, returnedFedCluster)
	}
}

func TestGetClustersByType(t *testing.T) {

	t.Run("get clusters by type", func(t *testing.T) {

		t.Run("not found", func(t *testing.T) {
			defer resetClusterCache()
			// given
			// empty cache

			//when
			clusters := clusterCache.getFedClustersByType(Member)

			//then
			assert.Empty(t, clusters)

			//when
			clusters = clusterCache.getFedClustersByType(Host)

			//then
			assert.Empty(t, clusters)
		})

		t.Run("found", func(t *testing.T) {
			defer resetClusterCache()
			// given
			// Two members, one host
			member1 := newTestFedCluster("cluster-1", Member, v1.ConditionTrue)
			clusterCache.addFedCluster(member1)
			member2 := newTestFedCluster("cluster-2", Member, v1.ConditionTrue)
			clusterCache.addFedCluster(member2)
			host := newTestFedCluster("cluster-3", Host, v1.ConditionTrue)
			clusterCache.addFedCluster(host)

			//when
			clusters := clusterCache.getFedClustersByType(Member)

			//then
			assert.Len(t, clusters, 2)
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)

			//when
			clusters = clusterCache.getFedClustersByType(Host)

			//then
			assert.Len(t, clusters, 1)
			assert.Contains(t, clusters, host)
		})
	})

	t.Run("get member clusters", func(t *testing.T) {
		defer resetClusterCache()

		// noise
		host := newTestFedCluster("cluster-host", Host, v1.ConditionTrue)
		clusterCache.addFedCluster(host)

		t.Run("not found", func(t *testing.T) {
			// given
			// no members

			//when
			clusters := GetMemberClusters()

			//then
			assert.Empty(t, clusters)
		})

		t.Run("found", func(t *testing.T) {
			// given
			member1 := newTestFedCluster("cluster-1", Member, v1.ConditionTrue)
			clusterCache.addFedCluster(member1)
			member2 := newTestFedCluster("cluster-2", Member, v1.ConditionTrue)
			clusterCache.addFedCluster(member2)

			//when
			clusters := GetMemberClusters()

			//then
			assert.Len(t, clusters, 2)
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)
		})
	})

	t.Run("get host cluster", func(t *testing.T) {
		defer resetClusterCache()

		// noise
		member1 := newTestFedCluster("cluster-member-1", Member, v1.ConditionTrue)
		clusterCache.addFedCluster(member1)

		t.Run("not found", func(t *testing.T) {
			// given
			// no host

			//when
			_, ok := GetHostCluster()

			//then
			assert.False(t, ok)
		})

		t.Run("found", func(t *testing.T) {
			// given
			host := newTestFedCluster("cluster-host", Host, v1.ConditionTrue)
			clusterCache.addFedCluster(host)

			//when
			cluster, ok := GetHostCluster()

			//then
			assert.True(t, ok)
			assert.Equal(t, host, cluster)
		})
	})
}

func TestGetClusterUsingDifferentKey(t *testing.T) {
	// given
	defer resetClusterCache()
	clusterCache.addFedCluster(newTestFedCluster("cluster", Member, v1.ConditionTrue))

	for _, getFedCluster := range getFedClusterFuncs {

		// when
		returnedFedCluster, ok := getFedCluster("testCluster")

		// then
		assert.False(t, ok)
		assert.Nil(t, returnedFedCluster)
	}
}

func TestUpdateCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	trueCluster := newTestFedCluster("testCluster", Member, v1.ConditionTrue)
	falseCluster := newTestFedCluster("testCluster", Member, v1.ConditionFalse)
	clusterCache.addFedCluster(trueCluster)

	// when
	clusterCache.addFedCluster(falseCluster)

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, falseCluster, clusterCache.clusters["testCluster"])
}

func TestDeleteCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	fedCluster := newTestFedCluster("testCluster", Member, v1.ConditionTrue)
	clusterCache.addFedCluster(fedCluster)
	clusterCache.addFedCluster(newTestFedCluster("cluster", Member, v1.ConditionTrue))
	assert.Len(t, clusterCache.clusters, 2)

	// when
	clusterCache.deleteFedCluster("cluster")

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, fedCluster, clusterCache.clusters["testCluster"])
}

func newTestFedCluster(name string, clusterType Type, status v1.ConditionStatus) *FedCluster {
	cl := fake.NewFakeClient()
	fedCluster := &FedCluster{
		Name:              name,
		Client:            cl,
		OperatorNamespace: name + "Namespace",
		Type:              clusterType,
		ClusterStatus: &v1beta1.KubeFedClusterStatus{
			Conditions: []v1beta1.ClusterCondition{{
				Type:   common.ClusterReady,
				Status: status,
			}},
		},
	}
	return fedCluster
}

func resetClusterCache() {
	clusterCache = kubeFedClusterClients{clusters: map[string]*FedCluster{}}
}
