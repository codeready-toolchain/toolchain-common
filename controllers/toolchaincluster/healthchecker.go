package toolchaincluster

import (
	"context"
	"strings"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	healthzOk              = "/healthz responded with ok"
	healthzNotOk           = "/healthz responded without ok"
	clusterNotReachableMsg = "cluster is not reachable"
	clusterReachableMsg    = "cluster is reachable"
)

type HealthChecker struct {
	localClusterClient     client.Client
	remoteClusterClient    client.Client
	remoteClusterClientset *kubeclientset.Clientset
	logger                 logr.Logger
}

func (hc *HealthChecker) updateIndividualClusterStatus(ctx context.Context, toolchainCluster *toolchainv1alpha1.ToolchainCluster) error {

	currentClusterStatus := hc.getClusterHealthStatus(ctx)

	for index, currentCond := range currentClusterStatus.Conditions {
		for _, previousCond := range toolchainCluster.Status.Conditions {
			if currentCond.Type == previousCond.Type && currentCond.Status == previousCond.Status {
				currentClusterStatus.Conditions[index].LastTransitionTime = previousCond.LastTransitionTime
			}
		}
	}

	toolchainCluster.Status = *currentClusterStatus
	if err := hc.localClusterClient.Status().Update(ctx, toolchainCluster); err != nil {
		return errors.Wrapf(err, "Failed to update the status of cluster %s", toolchainCluster.Name)
	}
	return nil
}

// getClusterHealthStatus gets the kubernetes cluster health status by requesting "/healthz"
func (hc *HealthChecker) getClusterHealthStatus(ctx context.Context) *toolchainv1alpha1.ToolchainClusterStatus {
	clusterStatus := toolchainv1alpha1.ToolchainClusterStatus{}
	body, err := hc.remoteClusterClientset.DiscoveryClient.RESTClient().Get().AbsPath("/healthz").Do(ctx).Raw()
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
		LastUpdatedTime:    &currentTime,
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
		LastUpdatedTime:    &currentTime,
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
		LastUpdatedTime:    &currentTime,
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
		LastUpdatedTime:    &currentTime,
		LastTransitionTime: &currentTime,
	}
}
