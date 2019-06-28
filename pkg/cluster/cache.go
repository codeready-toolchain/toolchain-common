package cluster

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sync"
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

func (c *kubeFedClusterClients) getFirstFedCluster() (*FedCluster, bool) {
	c.RLock()
	defer c.RUnlock()
	for _, cluster := range c.clusters {
		return cluster, true
	}
	return nil, false
}

// GetFedCluster returns a kube client for the cluster (with the given name) and info if the client exists
func GetFedCluster(name string) (*FedCluster, bool) {
	return clusterCache.getFedCluster(name)
}

// GetFirstFedCluster returns a first kube client from the cache of clusters and info if such a client exists
func GetFirstFedCluster() (*FedCluster, bool) {
	return clusterCache.getFirstFedCluster()
}

// Type is a cluster type (either host or member)
type Type string

const (
	Member Type = "member"
	Host   Type = "host"
)
