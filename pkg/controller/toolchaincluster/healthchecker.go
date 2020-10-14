package toolchaincluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclientset "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var logger = logf.Log.WithName("toolchaincluster_healthcheck")

const (
	healthzOk              = "/healthz responded with ok"
	healthzNotOk           = "/healthz responded without ok"
	clusterNotReachableMsg = "cluster is not reachable"
	clusterReachableMsg    = "cluster is reachable"
)

func StartHealthChecks(mgr manager.Manager, namespace string, stopChan <-chan struct{}, period time.Duration) {
	logger.Info("starting health checks", "period", period)
	go wait.Until(func() {
		updateClusterStatuses(namespace, mgr.GetClient())
	}, period, stopChan)
}

type HealthChecker struct {
	localClusterClient     client.Client
	remoteClusterClient    client.Client
	remoteClusterClientset *kubeclientset.Clientset
	logger                 logr.Logger
}

// updateClusterStatuses checks cluster health and updates status of all ToolchainClusters
func updateClusterStatuses(namespace string, cl client.Client) {
	clusters := &v1alpha1.ToolchainClusterList{}
	err := cl.List(context.TODO(), clusters, client.InNamespace(namespace))
	if err != nil {
		logger.Error(err, "unable to list existing ToolchainClusters")
		return
	}
	if len(clusters.Items) == 0 {
		logger.Info("no ToolchainCluster found")
	}

	for _, obj := range clusters.Items {
		clusterObj := obj.DeepCopy()
		clusterLogger := logger.WithValues("cluster-name", clusterObj.Name)

		cachedCluster, ok := cluster.GetCachedToolchainCluster(clusterObj.Name)
		if !ok {
			clusterLogger.Error(fmt.Errorf("cluster %s not found in cache", clusterObj.Name), "failed to retrieve stored data for cluster")
			clusterObj.Status.Conditions = []v1alpha1.ToolchainClusterCondition{clusterOfflineCondition()}
			if err := cl.Status().Update(context.TODO(), clusterObj); err != nil {
				clusterLogger.Error(err, "failed to update the status of ToolchainCluster")
			}
			continue
		}

		clientSet, err := kubeclientset.NewForConfig(cachedCluster.Config)
		if err != nil {
			clusterLogger.Error(err, "cannot create ClientSet for a ToolchainCluster")
			continue
		}

		healthChecker := &HealthChecker{
			localClusterClient:     cl,
			remoteClusterClient:    cachedCluster.Client,
			remoteClusterClientset: clientSet,
			logger:                 clusterLogger,
		}
		clusterLogger.Info("getting the current state of ToolchainCluster")
		if err := healthChecker.updateIndividualClusterStatus(clusterObj); err != nil {
			clusterLogger.Error(err, "unable to update cluster status of ToolchainCluster")
		}
	}
}

func (hc *HealthChecker) updateIndividualClusterStatus(toolchainCluster *v1alpha1.ToolchainCluster) error {

	currentClusterStatus := hc.getClusterHealthStatus()

	for index, currentCond := range currentClusterStatus.Conditions {
		for _, previousCond := range toolchainCluster.Status.Conditions {
			if currentCond.Type == previousCond.Type && currentCond.Status == previousCond.Status {
				currentClusterStatus.Conditions[index].LastTransitionTime = previousCond.LastTransitionTime
			}
		}
	}

	currentClusterStatus = hc.updateClusterZonesAndRegion(currentClusterStatus, toolchainCluster)

	toolchainCluster.Status = *currentClusterStatus
	if err := hc.localClusterClient.Status().Update(context.TODO(), toolchainCluster); err != nil {
		return errors.Wrapf(err, "Failed to update the status of cluster %s", toolchainCluster.Name)
	}
	return nil
}

func (hc *HealthChecker) updateClusterZonesAndRegion(currentClusterStatus *v1alpha1.ToolchainClusterStatus, toolchainCluster *v1alpha1.ToolchainCluster) *v1alpha1.ToolchainClusterStatus {
	if !cluster.IsReady(currentClusterStatus) {
		return currentClusterStatus
	}

	zones, region, err := hc.getClusterZones()
	if err != nil {
		hc.logger.Error(err, "Failed to get zones and region for the cluster")
		return currentClusterStatus
	}

	// If new zone & region are empty, preserve the old ones so that user configured zone & region
	// labels are effective
	if len(zones) == 0 {
		zones = toolchainCluster.Status.Zones
	}
	if len(region) == 0 && toolchainCluster.Status.Region != nil {
		region = *toolchainCluster.Status.Region
	}
	currentClusterStatus.Zones = zones
	currentClusterStatus.Region = &region
	return currentClusterStatus
}

// getClusterHealthStatus gets the kubernetes cluster health status by requesting "/healthz"
func (hc *HealthChecker) getClusterHealthStatus() *v1alpha1.ToolchainClusterStatus {
	clusterStatus := v1alpha1.ToolchainClusterStatus{}
	body, err := hc.remoteClusterClientset.DiscoveryClient.RESTClient().Get().AbsPath("/healthz").Do(context.TODO()).Raw()
	if err != nil {
		hc.logger.Error(err, "Failed to do cluster health check for a ToolchainCluster")
		clusterStatus.Conditions = append(clusterStatus.Conditions, clusterOfflineCondition())
	} else {
		if !strings.EqualFold(string(body), "ok") {
			clusterStatus.Conditions = append(clusterStatus.Conditions, clusterNotReadyCondition(), clusterNotOfflineCondition())
		} else {
			clusterStatus.Conditions = append(clusterStatus.Conditions, clusterReadyCondition())
		}
	}

	return &clusterStatus
}

func clusterReadyCondition() v1alpha1.ToolchainClusterCondition {
	currentTime := metav1.Now()
	return v1alpha1.ToolchainClusterCondition{
		Type:               v1alpha1.ToolchainClusterReady,
		Status:             corev1.ConditionTrue,
		Reason:             v1alpha1.ToolchainClusterClusterReadyReason,
		Message:            healthzOk,
		LastProbeTime:      currentTime,
		LastTransitionTime: &currentTime,
	}
}

func clusterNotReadyCondition() v1alpha1.ToolchainClusterCondition {
	currentTime := metav1.Now()
	return v1alpha1.ToolchainClusterCondition{
		Type:               v1alpha1.ToolchainClusterReady,
		Status:             corev1.ConditionFalse,
		Reason:             v1alpha1.ToolchainClusterClusterNotReadyReason,
		Message:            healthzNotOk,
		LastProbeTime:      currentTime,
		LastTransitionTime: &currentTime,
	}
}

func clusterOfflineCondition() v1alpha1.ToolchainClusterCondition {
	currentTime := metav1.Now()
	return v1alpha1.ToolchainClusterCondition{
		Type:               v1alpha1.ToolchainClusterOffline,
		Status:             corev1.ConditionTrue,
		Reason:             v1alpha1.ToolchainClusterClusterNotReachableReason,
		Message:            clusterNotReachableMsg,
		LastProbeTime:      currentTime,
		LastTransitionTime: &currentTime,
	}
}

func clusterNotOfflineCondition() v1alpha1.ToolchainClusterCondition {
	currentTime := metav1.Now()
	return v1alpha1.ToolchainClusterCondition{
		Type:               v1alpha1.ToolchainClusterOffline,
		Status:             corev1.ConditionFalse,
		Reason:             v1alpha1.ToolchainClusterClusterReachableReason,
		Message:            clusterReachableMsg,
		LastProbeTime:      currentTime,
		LastTransitionTime: &currentTime,
	}
}

// getClusterZones gets the kubernetes cluster zones and region by inspecting labels on nodes in the cluster.
func (hc *HealthChecker) getClusterZones() ([]string, string, error) {
	nodes := &corev1.NodeList{}
	err := hc.remoteClusterClient.List(context.TODO(), nodes)
	if err != nil {
		return nil, "", err
	}

	zones := sets.NewString()
	region := ""
	for i, node := range nodes.Items {
		zone := getZoneNameForNode(node)
		// region is same for all nodes in the cluster, so just pick the region from first node.
		if i == 0 {
			region = getRegionNameForNode(node)
		}
		if zone != "" && !zones.Has(zone) {
			zones.Insert(zone)
		}
	}
	return zones.List(), region, nil
}

// Find the name of the zone in which a Node is running.
func getZoneNameForNode(node corev1.Node) string {
	for key, value := range node.Labels {
		if key == corev1.LabelZoneFailureDomain {
			return value
		}
	}
	return ""
}

// Find the name of the region in which a Node is running.
func getRegionNameForNode(node corev1.Node) string {
	for key, value := range node.Labels {
		if key == corev1.LabelZoneRegion {
			return value
		}
	}
	return ""
}
