package cluster

import (
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

var clusterCache = kubeFedClusterClients{clusters: map[string]*FedCluster{}}

type kubeFedClusterClients struct {
	sync.RWMutex
	clusters map[string]*FedCluster
}

// FedCluster stores cluster client; cluster related info and previous health check probe results
type FedCluster struct {
	// Client is the kube client for the cluster.
	Client client.Client
	// Name is a name of the cluster. Has to be unique - is used as a key in a map.
	Name string
	// Type is a type of the cluster (either host or member)
	Type Type
	// OperatorNamespace is a name of a namespace (in the cluster) the operator is running in
	OperatorNamespace string
	// ClusterStatus is the cluster result as of the last health check probe.
	ClusterStatus *v1beta1.KubeFedClusterStatus
	// OwnerClusterName keeps the name of the cluster the KubeFedCluster resource is created in
	// eg. if this KubeFedCluster identifies a Host cluster (and thus is created in Member)
	// then the OwnerClusterName has a name of the member - it has to be same name as the name
	// that is used for identifying the member in a Host cluster
	OwnerClusterName string
}

func (c *kubeFedClusterClients) addFedCluster(cluster *FedCluster) {
	c.Lock()
	defer c.Unlock()
	c.clusters[cluster.Name] = cluster
}

func (c *kubeFedClusterClients) deleteFedCluster(name string) {
	c.Lock()
	defer c.Unlock()
	delete(c.clusters, name)
}

func (c *kubeFedClusterClients) getFedCluster(name string) (*FedCluster, bool) {
	c.RLock()
	defer c.RUnlock()
	cluster, ok := c.clusters[name]
	return cluster, ok
}

func (c *kubeFedClusterClients) getFedClustersByType(clusterType Type) []*FedCluster {
	c.RLock()
	defer c.RUnlock()
	clusters := make([]*FedCluster, 0, len(c.clusters))
	for _, cluster := range c.clusters {
		if cluster.Type == clusterType {
			clusters = append(clusters, cluster)
		}
	}
	return clusters
}

// GetFedCluster returns a kube client for the cluster (with the given name) and info if the client exists
func GetFedCluster(name string) (*FedCluster, bool) {
	return clusterCache.getFedCluster(name)
}

// GetHostClusterFunc a func that returns the Host cluster from the cache,
// along with a bool to indicate if there was a match or not
type GetHostClusterFunc func() (*FedCluster, bool)

// HostCluster the func to retrieve the host cluster
var HostCluster GetHostClusterFunc = GetHostCluster

// GetHostCluster returns the kube client for the host cluster from the cache of the clusters
// and info if such a client exists
func GetHostCluster() (*FedCluster, bool) {
	clusters := clusterCache.getFedClustersByType(Host)
	if len(clusters) == 0 {
		return nil, false
	}
	return clusters[0], true
}

// GetMemberClustersFunc a func that returns the member clusters from the cache
type GetMemberClustersFunc func() []*FedCluster

// MemberClusters the func to retrieve the member clusters
var MemberClusters GetMemberClustersFunc = GetMemberClusters

// GetMemberClusters returns the kube clients for the host clusters from the cache of the clusters
func GetMemberClusters() []*FedCluster {
	return clusterCache.getFedClustersByType(Member)
}

// Type is a cluster type (either host or member)
type Type string

const (
	Member Type = "member"
	Host   Type = "host"
)
