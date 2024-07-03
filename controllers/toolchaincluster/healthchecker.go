package toolchaincluster

import (
	"context"
	"strings"

	kubeclientset "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	healthzOk              = "/healthz responded with ok"
	healthzNotOk           = "/healthz responded without ok"
	clusterNotReachableMsg = "cluster is not reachable"
	clusterReachableMsg    = "cluster is reachable"
)

// getClusterHealth gets the kubernetes cluster health status by requesting "/healthz"
func getClusterHealthStatus(ctx context.Context, remoteClusterClientset *kubeclientset.Clientset) (bool, error) {

	lgr := log.FromContext(ctx)
	body, err := remoteClusterClientset.DiscoveryClient.RESTClient().Get().AbsPath("/healthz").Do(ctx).Raw()
	if err != nil {
		lgr.Error(err, "Failed to do cluster health check for a ToolchainCluster")
		return false, err
	}
	return strings.EqualFold(string(body), "ok"), nil

}
