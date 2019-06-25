package cluster

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"testing"
)

func TestAddCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	fedCluster := newTestFedCluster("testCluster", v1.ConditionTrue)

	// when
	clusterCache.addFedCluster(fedCluster)

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, fedCluster, clusterCache.clusters["testCluster"])
}

func TestGetCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	fedCluster := newTestFedCluster("testCluster", v1.ConditionTrue)
	clusterCache.addFedCluster(fedCluster)
	clusterCache.addFedCluster(newTestFedCluster("cluster", v1.ConditionTrue))

	// when
	returnedFedCluster, ok := clusterCache.getFedCluster("testCluster")

	// then
	assert.True(t, ok)
	assert.Equal(t, fedCluster, returnedFedCluster)
}

func TestGetClusterWhenIsEmpty(t *testing.T) {
	// given
	resetClusterCache()

	// when
	returnedFedCluster, ok := clusterCache.getFedCluster("testCluster")

	// then
	assert.False(t, ok)
	assert.Nil(t, returnedFedCluster)
}

func TestGetClusterUsingDifferentKey(t *testing.T) {
	// given
	defer resetClusterCache()
	clusterCache.addFedCluster(newTestFedCluster("cluster", v1.ConditionTrue))

	// when
	returnedFedCluster, ok := clusterCache.getFedCluster("testCluster")

	// then
	assert.False(t, ok)
	assert.Nil(t, returnedFedCluster)
}

func TestExportedGetCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	fedCluster := newTestFedCluster("testCluster", v1.ConditionTrue)
	clusterCache.addFedCluster(fedCluster)
	clusterCache.addFedCluster(newTestFedCluster("cluster", v1.ConditionTrue))

	// when
	returnedFedCluster, ok := GetFedCluster("testCluster")

	// then
	assert.True(t, ok)
	assert.Equal(t, fedCluster, returnedFedCluster)
}

func TestUpdateCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	trueCluster := newTestFedCluster("testCluster", v1.ConditionTrue)
	falseCluster := newTestFedCluster("testCluster", v1.ConditionFalse)
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
	fedCluster := newTestFedCluster("testCluster", v1.ConditionTrue)
	clusterCache.addFedCluster(fedCluster)
	clusterCache.addFedCluster(newTestFedCluster("cluster", v1.ConditionTrue))
	assert.Len(t, clusterCache.clusters, 2)

	// when
	clusterCache.deleteFedCluster("cluster")

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, fedCluster, clusterCache.clusters["testCluster"])
}

func newTestFedCluster(name string, status v1.ConditionStatus) *FedCluster {
	cl := fake.NewFakeClient()
	fedCluster := &FedCluster{
		Name:              name,
		Client:            cl,
		OperatorNamespace: name + "Namespace",
		Type:              Member,
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
