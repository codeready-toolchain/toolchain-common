package cluster

import (
	"sync"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func getOrFetchCachedToolchainCluster() func(name string) (*CachedToolchainCluster, bool) {
	return func(name string) (cluster *CachedToolchainCluster, b bool) {
		return clusterCache.getCachedToolchainCluster(name, true)
	}
}

var getCachedToolchainClusterFuncs = []func(name string) (*CachedToolchainCluster, bool){
	getOrFetchCachedToolchainCluster(), GetCachedToolchainCluster}

func TestAddCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	cachedCluster := newTestCachedToolchainCluster(t, "testCluster", ready)

	// when
	clusterCache.addCachedToolchainCluster(cachedCluster)

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, cachedCluster, clusterCache.clusters["testCluster"])
}

func TestGetCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	cachedCluster := newTestCachedToolchainCluster(t, "testCluster", ready)
	clusterCache.addCachedToolchainCluster(cachedCluster)
	clusterCache.addCachedToolchainCluster(newTestCachedToolchainCluster(t, "cluster", ready))

	for _, getCachedCluster := range getCachedToolchainClusterFuncs {

		// when
		returnedCachedCluster, ok := getCachedCluster("testCluster")

		// then
		assert.True(t, ok)
		assert.Equal(t, cachedCluster, returnedCachedCluster)
	}
}

func TestGetClusterWhenIsEmpty(t *testing.T) {
	// given
	resetClusterCache()

	for _, getCachedCluster := range getCachedToolchainClusterFuncs {

		// when
		returnedCachedCluster, ok := getCachedCluster("testCluster")

		// then
		assert.False(t, ok)
		assert.Nil(t, returnedCachedCluster)
	}
}

func TestGetClusters(t *testing.T) {

	t.Run("get clusters", func(t *testing.T) {

		t.Run("not found", func(t *testing.T) {
			// given
			defer resetClusterCache()
			// no clusters

			//when
			clusters := GetClusters()

			//then
			assert.Empty(t, clusters)
		})

		t.Run("all clusters", func(t *testing.T) {
			// given
			defer resetClusterCache()
			host := newTestCachedToolchainCluster(t, "cluster-host", ready)
			clusterCache.addCachedToolchainCluster(host)
			member1 := newTestCachedToolchainCluster(t, "cluster-1", ready)
			clusterCache.addCachedToolchainCluster(member1)
			member2 := newTestCachedToolchainCluster(t, "cluster-2", ready)
			clusterCache.addCachedToolchainCluster(member2)

			//when
			clusters := GetClusters()

			//then
			assert.Len(t, clusters, 3)
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)
			assert.Contains(t, clusters, host)
		})

		t.Run("found after refreshing the cache", func(t *testing.T) {
			// given
			defer resetClusterCache()
			clustertest := newTestCachedToolchainCluster(t, "clustertotest", ready)
			called := false
			clusterCache.refreshCache = func() {
				called = true
				clusterCache.addCachedToolchainCluster(clustertest)
			}

			//when
			clusters := GetClusters()

			//then
			assert.Len(t, clusters, 1)
			assert.Contains(t, clusters, clustertest)
			assert.True(t, called)
		})

		t.Run("get clusters filtered by readiness and capacity", func(t *testing.T) {
			defer resetClusterCache()

			host := newTestCachedToolchainCluster(t, "cluster-host", ready)
			clusterCache.addCachedToolchainCluster(host)
			member1 := newTestCachedToolchainCluster(t, "cluster-1", ready)
			clusterCache.addCachedToolchainCluster(member1)
			member2 := newTestCachedToolchainCluster(t, "cluster-2", ready)
			clusterCache.addCachedToolchainCluster(member2)
			// noise
			member3 := newTestCachedToolchainCluster(t, "cluster-3", notReady)
			clusterCache.addCachedToolchainCluster(member3)

			//when
			clusters := GetClusters(Ready)

			//then
			assert.Len(t, clusters, 3)
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)
			assert.Contains(t, clusters, host)

		})

	})

}

func TestGetClusterUsingDifferentKey(t *testing.T) {
	// given
	defer resetClusterCache()
	clusterCache.addCachedToolchainCluster(newTestCachedToolchainCluster(t, "cluster", ready))

	for _, getCachedCluster := range getCachedToolchainClusterFuncs {

		// when
		returnedCachedCluster, ok := getCachedCluster("testCluster")

		// then
		assert.False(t, ok)
		assert.Nil(t, returnedCachedCluster)
	}
}

func TestUpdateCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	trueCluster := newTestCachedToolchainCluster(t, "testCluster", ready)
	falseCluster := newTestCachedToolchainCluster(t, "testCluster", notReady)
	clusterCache.addCachedToolchainCluster(trueCluster)

	// when
	clusterCache.addCachedToolchainCluster(falseCluster)

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, falseCluster, clusterCache.clusters["testCluster"])
}

func TestDeleteCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	cachedCluster := newTestCachedToolchainCluster(t, "testCluster", ready)
	clusterCache.addCachedToolchainCluster(cachedCluster)
	clusterCache.addCachedToolchainCluster(newTestCachedToolchainCluster(t, "cluster", ready))
	assert.Len(t, clusterCache.clusters, 2)

	// when
	clusterCache.deleteCachedToolchainCluster("cluster")

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, cachedCluster, clusterCache.clusters["testCluster"])
}

func TestRefreshCache(t *testing.T) {
	// given
	testCluster := newTestCachedToolchainCluster(t, "testCluster", ready)
	newCluster := newTestCachedToolchainCluster(t, "newCluster", ready)

	t.Run("refresh enabled", func(t *testing.T) {
		defer resetClusterCache()
		clusterCache.addCachedToolchainCluster(testCluster)
		clusterCache.refreshCache = func() {
			clusterCache.addCachedToolchainCluster(newCluster)
		}
		t.Run("refresh and get existing cluster", func(t *testing.T) {
			// when
			returnedNewCluster, ok := clusterCache.getCachedToolchainCluster("newCluster", true)

			// then
			assert.True(t, ok)
			assert.Equal(t, newCluster, returnedNewCluster)

			returnedTestCluster, ok := clusterCache.getCachedToolchainCluster("testCluster", true)
			assert.True(t, ok)
			assert.Equal(t, testCluster, returnedTestCluster)
		})

		t.Run("refresh and get non-existing cluster", func(t *testing.T) {
			// when
			cluster, ok := clusterCache.getCachedToolchainCluster("anotherCluster", true)

			// then
			assert.False(t, ok)
			assert.Nil(t, cluster)
		})
	})

	t.Run("refresh disabled", func(t *testing.T) {
		defer resetClusterCache()
		clusterCache.addCachedToolchainCluster(testCluster)
		clusterCache.refreshCache = func() {
			clusterCache.addCachedToolchainCluster(newCluster)
		}
		t.Run("don't refresh and get the only cluster that is in cache", func(t *testing.T) {
			// when
			returnedNewCluster, ok := clusterCache.getCachedToolchainCluster("newCluster", false)

			// then
			assert.False(t, ok)
			assert.Nil(t, returnedNewCluster)

			returnedTestCluster, ok := clusterCache.getCachedToolchainCluster("testCluster", false)
			assert.True(t, ok)
			assert.Equal(t, testCluster, returnedTestCluster)
		})

		t.Run("non-existing cluster", func(t *testing.T) {
			// when
			cluster, ok := clusterCache.getCachedToolchainCluster("anotherCluster", false)

			// then
			assert.False(t, ok)
			assert.Nil(t, cluster)
		})
	})
}

func TestMultipleActionsInParallel(t *testing.T) {
	// given
	clusterForTest := newTestCachedToolchainCluster(t, "clusterForTest", ready)

	defer resetClusterCache()
	var latch sync.WaitGroup
	latch.Add(1)
	var waitForFinished sync.WaitGroup

	clusterCache.refreshCache = func() {
		clusterCache.addCachedToolchainCluster(clusterForTest)

	}

	for i := 0; i < 1000; i++ {
		waitForFinished.Add(4)
		go func() {
			defer waitForFinished.Done()
			latch.Wait()
			clusterCache.addCachedToolchainCluster(clusterForTest)
		}()
		go func() {
			defer waitForFinished.Done()
			latch.Wait()
			cluster, ok := clusterCache.getCachedToolchainCluster(clusterForTest.Name, true)
			if ok {
				assert.Equal(t, clusterForTest, cluster)
			} else {
				assert.Nil(t, cluster)
			}
		}()
		go func() {
			defer waitForFinished.Done()
			latch.Wait()
			clusters := clusterCache.getCachedToolchainClusters()
			if len(clusters) == 1 {
				assert.Equal(t, clusterForTest, clusters[0])
			} else {
				assert.Empty(t, clusters)
			}
		}()
		go func(clusterToTest *CachedToolchainCluster) {
			defer waitForFinished.Done()
			latch.Wait()
			clusterCache.deleteCachedToolchainCluster(clusterToTest.Name)
		}(clusterForTest)
	}

	// when
	latch.Done()

	// then
	waitForFinished.Wait()

	clusterForTest1, ok := clusterCache.getCachedToolchainCluster("clusterForTest", true)
	assert.True(t, ok)
	assert.Equal(t, clusterForTest, clusterForTest1)
}

// clusterOption an option to configure the cluster to use in the tests
type clusterOption func(*CachedToolchainCluster)

// Ready an option to state the cluster as "ready"
var ready clusterOption = func(c *CachedToolchainCluster) {
	c.ClusterStatus.Conditions = append(c.ClusterStatus.Conditions, toolchainv1alpha1.ToolchainClusterCondition{
		Type:   toolchainv1alpha1.ToolchainClusterReady,
		Status: v1.ConditionTrue,
	})
}

// clusterNotReady an option to state the cluster as "not ready"
var notReady clusterOption = func(c *CachedToolchainCluster) {
	c.ClusterStatus.Conditions = append(c.ClusterStatus.Conditions, toolchainv1alpha1.ToolchainClusterCondition{
		Type:   toolchainv1alpha1.ToolchainClusterReady,
		Status: v1.ConditionFalse,
	})
}

func newTestCachedToolchainCluster(t *testing.T, name string, options ...clusterOption) *CachedToolchainCluster {
	cl := test.NewFakeClient(t)
	cachedCluster := &CachedToolchainCluster{
		Config: &Config{
			Name:              name,
			OperatorNamespace: name + "Namespace",
		},
		Client:        cl,
		ClusterStatus: &toolchainv1alpha1.ToolchainClusterStatus{},
	}
	for _, configure := range options {
		configure(cachedCluster)
	}
	return cachedCluster
}

func resetClusterCache() {
	clusterCache = toolchainClusterClients{clusters: map[string]*CachedToolchainCluster{}}
}
