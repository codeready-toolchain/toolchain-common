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

// GetFedCluster returns a kube client for the cluster (with the given name) and info if the client exists
func GetFedCluster(name string) (*FedCluster, bool) {
	return clusterCache.getFedCluster(name)
}

// Type is a cluster type (either host or member)
type Type string

const (
	Member Type = "member"
	Host   Type = "host"
)
