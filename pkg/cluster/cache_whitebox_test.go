package cluster

import (
	"sync"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHostCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	host := newTestCachedToolchainCluster(t, "host-cluster", ready)
	clusterCache.addCachedToolchainCluster(host)

	// when
	returnedCachedCluster, ok := HostCluster()

	// then
	assert.True(t, ok)
	assert.Equal(t, host, returnedCachedCluster)
}

func TestMemberClusters(t *testing.T) {
	// given
	defer resetClusterCache()
	member1 := newTestCachedToolchainCluster(t, "member-cluster-1", ready)
	clusterCache.addCachedToolchainCluster(member1)
	member2 := newTestCachedToolchainCluster(t, "member-cluster-2", ready)
	clusterCache.addCachedToolchainCluster(member2)

	// when
	returnedCachedClusters := MemberClusters()

	// then
	require.Len(t, returnedCachedClusters, 2)
	assert.Contains(t, returnedCachedClusters, member1)
	assert.Contains(t, returnedCachedClusters, member2)
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

func TestGetClustersByType(t *testing.T) {

	t.Run("get clusters by type", func(t *testing.T) {

		t.Run("not found", func(t *testing.T) {
			defer resetClusterCache()
			// given
			// empty cache

			//when
			clusters := clusterCache.getCachedToolchainClustersByType()

			//then
			assert.Empty(t, clusters)

			//when
			clusters = clusterCache.getCachedToolchainClustersByType()

			//then
			assert.Empty(t, clusters)
		})

		t.Run("found", func(t *testing.T) {
			defer resetClusterCache()
			// given
			// Two members, one host
			member1 := newTestCachedToolchainCluster(t, "cluster-1", ready)
			clusterCache.addCachedToolchainCluster(member1)
			member2 := newTestCachedToolchainCluster(t, "cluster-2", ready)
			clusterCache.addCachedToolchainCluster(member2)
			host := newTestCachedToolchainCluster(t, "cluster-3", ready)
			clusterCache.addCachedToolchainCluster(host)

			//when
			clusters := clusterCache.getCachedToolchainClustersByType()

			//then
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)

			//when
			clusters = clusterCache.getCachedToolchainClustersByType()

			//then
			assert.Contains(t, clusters, host)
		})
	})

	t.Run("get member clusters", func(t *testing.T) {

		t.Run("all clusters", func(t *testing.T) {
			// given
			defer resetClusterCache()
			member1 := newTestCachedToolchainCluster(t, "cluster-1", ready)
			clusterCache.addCachedToolchainCluster(member1)
			member2 := newTestCachedToolchainCluster(t, "cluster-2", ready)
			clusterCache.addCachedToolchainCluster(member2)

			//when
			clusters := GetMemberClusters()

			//then
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)
		})

		t.Run("found after refreshing the cache", func(t *testing.T) {
			// given
			defer resetClusterCache()
			member := newTestCachedToolchainCluster(t, "member", ready)
			called := false
			clusterCache.refreshCache = func() {
				called = true
				clusterCache.addCachedToolchainCluster(member)
			}

			//when
			clusters := GetMemberClusters()

			//then
			assert.Contains(t, clusters, member)
			assert.True(t, called)
		})

	})

	t.Run("get member clusters filtered by readiness and capacity", func(t *testing.T) {
		defer resetClusterCache()

		// noise
		host := newTestCachedToolchainCluster(t, "cluster-host", ready)
		clusterCache.addCachedToolchainCluster(host)
		member1 := newTestCachedToolchainCluster(t, "cluster-1", ready)
		clusterCache.addCachedToolchainCluster(member1)
		member2 := newTestCachedToolchainCluster(t, "cluster-2", ready)
		clusterCache.addCachedToolchainCluster(member2)
		member3 := newTestCachedToolchainCluster(t, "cluster-3", notReady)
		clusterCache.addCachedToolchainCluster(member3)

		t.Run("get only ready member clusters", func(t *testing.T) {
			//when
			clusters := GetMemberClusters(Ready)

			//then
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)
		})
	})

	t.Run("get host cluster", func(t *testing.T) {

		t.Run("found", func(t *testing.T) {
			// given
			defer resetClusterCache()
			host := newTestCachedToolchainCluster(t, "cluster-host", ready)
			clusterCache.addCachedToolchainCluster(host)

			//when
			cluster, ok := GetHostCluster()

			//then
			assert.True(t, ok)
			assert.Equal(t, host, cluster)
		})

		t.Run("found after refreshing the cache", func(t *testing.T) {
			// given
			defer resetClusterCache()
			host := newTestCachedToolchainCluster(t, "cluster-host", ready)
			called := false
			clusterCache.refreshCache = func() {
				called = true
				clusterCache.addCachedToolchainCluster(host)
			}

			//when
			cluster, ok := GetHostCluster()

			//then
			assert.True(t, ok)
			assert.Equal(t, host, cluster)
			assert.True(t, called)
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
	defer resetClusterCache()
	var latch sync.WaitGroup
	latch.Add(1)
	var waitForFinished sync.WaitGroup

	memberCluster := newTestCachedToolchainCluster(t, "memberCluster", ready)
	hostCluster := newTestCachedToolchainCluster(t, "hostCluster", ready)
	clusterCache.refreshCache = func() {
		clusterCache.addCachedToolchainCluster(memberCluster)
		clusterCache.addCachedToolchainCluster(hostCluster)
	}

	for _, clusterToTest := range []*CachedToolchainCluster{memberCluster, hostCluster} {
		for i := 0; i < 1000; i++ {
			waitForFinished.Add(4)
			go func() {
				defer waitForFinished.Done()
				latch.Wait()
				clusterCache.addCachedToolchainCluster(clusterToTest)
			}()
			go func() {
				defer waitForFinished.Done()
				latch.Wait()
				cluster, ok := clusterCache.getCachedToolchainCluster(clusterToTest.Name, true)
				if ok {
					assert.Equal(t, clusterToTest, cluster)
				} else {
					assert.Nil(t, cluster)
				}
			}()
			go func() {
				defer waitForFinished.Done()
				latch.Wait()
				clusters := clusterCache.getCachedToolchainClustersByType()
				if len(clusters) == 1 {
					assert.Equal(t, clusterToTest, clusters[0])
				} else {
					assert.Empty(t, clusters)
				}
			}()
			go func(clusterToTest *CachedToolchainCluster) {
				defer waitForFinished.Done()
				latch.Wait()
				clusterCache.deleteCachedToolchainCluster(clusterToTest.Name)
			}(clusterToTest)
		}
	}

	// when
	latch.Done()

	// then
	waitForFinished.Wait()

	member, ok := clusterCache.getCachedToolchainCluster("memberCluster", true)
	assert.True(t, ok)
	assert.Equal(t, memberCluster, member)

	host, ok := clusterCache.getCachedToolchainCluster("hostCluster", true)
	assert.True(t, ok)
	assert.Equal(t, hostCluster, host)
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
