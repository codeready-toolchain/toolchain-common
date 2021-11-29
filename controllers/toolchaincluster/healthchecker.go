package toolchaincluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func StartHealthChecks(ctx context.Context, mgr manager.Manager, namespace string, period time.Duration) {
	logger.Info("starting health checks", "period", period)
	go wait.Until(func() {
		updateClusterStatuses(namespace, mgr.GetClient())
	}, period, ctx.Done())
}

type HealthChecker struct {
	localClusterClient     client.Client
	remoteClusterClient    client.Client
	remoteClusterClientset *kubeclientset.Clientset
	logger                 logr.Logger
}

// updateClusterStatuses checks cluster health and updates status of all ToolchainClusters
func updateClusterStatuses(namespace string, cl client.Client) {
	clusters := &toolchainv1alpha1.ToolchainClusterList{}
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
			clusterObj.Status.Conditions = []toolchainv1alpha1.ToolchainClusterCondition{clusterOfflineCondition()}
			if err := cl.Status().Update(context.TODO(), clusterObj); err != nil {
				clusterLogger.Error(err, "failed to update the status of ToolchainCluster")
			}
			continue
		}

		clientSet, err := kubeclientset.NewForConfig(cachedCluster.RestConfig)
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

func (hc *HealthChecker) updateIndividualClusterStatus(toolchainCluster *toolchainv1alpha1.ToolchainCluster) error {

	currentClusterStatus := hc.getClusterHealthStatus()

	for index, currentCond := range currentClusterStatus.Conditions {
		for _, previousCond := range toolchainCluster.Status.Conditions {
			if currentCond.Type == previousCond.Type && currentCond.Status == previousCond.Status {
				currentClusterStatus.Conditions[index].LastTransitionTime = previousCond.LastTransitionTime
			}
		}
	}

	toolchainCluster.Status = *currentClusterStatus
	if err := hc.localClusterClient.Status().Update(context.TODO(), toolchainCluster); err != nil {
		return errors.Wrapf(err, "Failed to update the status of cluster %s", toolchainCluster.Name)
	}
	return nil
}

// getClusterHealthStatus gets the kubernetes cluster health status by requesting "/healthz"
func (hc *HealthChecker) getClusterHealthStatus() *toolchainv1alpha1.ToolchainClusterStatus {
	clusterStatus := toolchainv1alpha1.ToolchainClusterStatus{}
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

func clusterReadyCondition() toolchainv1alpha1.ToolchainClusterCondition {
	currentTime := metav1.Now()
	return toolchainv1alpha1.ToolchainClusterCondition{
		Type:               toolchainv1alpha1.ToolchainClusterReady,
		Status:             corev1.ConditionTrue,
		Reason:             toolchainv1alpha1.ToolchainClusterClusterReadyReason,
		Message:            healthzOk,
		LastProbeTime:      currentTime,
		LastTransitionTime: &currentTime,
	}
}

func clusterNotReadyCondition() toolchainv1alpha1.ToolchainClusterCondition {
	currentTime := metav1.Now()
	return toolchainv1alpha1.ToolchainClusterCondition{
		Type:               toolchainv1alpha1.ToolchainClusterReady,
		Status:             corev1.ConditionFalse,
		Reason:             toolchainv1alpha1.ToolchainClusterClusterNotReadyReason,
		Message:            healthzNotOk,
		LastProbeTime:      currentTime,
		LastTransitionTime: &currentTime,
	}
}

func clusterOfflineCondition() toolchainv1alpha1.ToolchainClusterCondition {
	currentTime := metav1.Now()
	return toolchainv1alpha1.ToolchainClusterCondition{
		Type:               toolchainv1alpha1.ToolchainClusterOffline,
		Status:             corev1.ConditionTrue,
		Reason:             toolchainv1alpha1.ToolchainClusterClusterNotReachableReason,
		Message:            clusterNotReachableMsg,
		LastProbeTime:      currentTime,
		LastTransitionTime: &currentTime,
	}
}

func clusterNotOfflineCondition() toolchainv1alpha1.ToolchainClusterCondition {
	currentTime := metav1.Now()
	return toolchainv1alpha1.ToolchainClusterCondition{
		Type:               toolchainv1alpha1.ToolchainClusterOffline,
		Status:             corev1.ConditionFalse,
		Reason:             toolchainv1alpha1.ToolchainClusterClusterReachableReason,
		Message:            clusterReachableMsg,
		LastProbeTime:      currentTime,
		LastTransitionTime: &currentTime,
	}
}
