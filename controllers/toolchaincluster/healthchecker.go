package toolchaincluster

import (
	"context"
	"strings"

	"github.com/codeready-toolchain/api/api/v1alpha1"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	healthzOk              = "/healthz responded with ok"
	healthzNotOk           = "/healthz responded without ok"
	clusterNotReachableMsg = "cluster is not reachable"
	clusterReachableMsg    = "cluster is reachable"
)

// getClusterHealthStatus gets the kubernetes cluster health status by requesting "/healthz"
func getClusterHealthStatus(ctx context.Context, remoteClusterClientset *kubeclientset.Clientset) []v1alpha1.Condition {
	lgr := log.FromContext(ctx)
	conditions := []v1alpha1.Condition{}
	body, err := remoteClusterClientset.DiscoveryClient.RESTClient().Get().AbsPath("/healthz").Do(ctx).Raw()
	if err != nil {
		lgr.Error(err, "Failed to do cluster health check for a ToolchainCluster")
		conditions = append(conditions, clusterOfflineCondition())
	} else {
		if !strings.EqualFold(string(body), "ok") {
			conditions = append(conditions, clusterNotReadyCondition(), clusterNotOfflineCondition())
		} else {
			conditions = append(conditions, clusterReadyCondition())
		}
	}
	return conditions
}

func clusterReadyCondition() toolchainv1alpha1.Condition {
	currentTime := metav1.Now()
	return toolchainv1alpha1.Condition{
		Type:               toolchainv1alpha1.ConditionReady,
		Status:             corev1.ConditionTrue,
		Reason:             toolchainv1alpha1.ToolchainClusterClusterReadyReason,
		Message:            healthzOk,
		LastUpdatedTime:    &currentTime,
		LastTransitionTime: currentTime,
	}
}

func clusterNotReadyCondition() toolchainv1alpha1.Condition {
	currentTime := metav1.Now()
	return toolchainv1alpha1.Condition{
		Type:               toolchainv1alpha1.ConditionReady,
		Status:             corev1.ConditionFalse,
		Reason:             toolchainv1alpha1.ToolchainClusterClusterNotReadyReason,
		Message:            healthzNotOk,
		LastUpdatedTime:    &currentTime,
		LastTransitionTime: currentTime,
	}
}

func clusterOfflineCondition() toolchainv1alpha1.Condition {
	currentTime := metav1.Now()
	return toolchainv1alpha1.Condition{
		Type:               toolchainv1alpha1.ToolchainClusterOffline,
		Status:             corev1.ConditionTrue,
		Reason:             toolchainv1alpha1.ToolchainClusterClusterNotReachableReason,
		Message:            clusterNotReachableMsg,
		LastUpdatedTime:    &currentTime,
		LastTransitionTime: currentTime,
	}
}

func clusterNotOfflineCondition() toolchainv1alpha1.Condition {
	currentTime := metav1.Now()
	return toolchainv1alpha1.Condition{
		Type:               toolchainv1alpha1.ToolchainClusterOffline,
		Status:             corev1.ConditionFalse,
		Reason:             toolchainv1alpha1.ToolchainClusterClusterReachableReason,
		Message:            clusterReachableMsg,
		LastUpdatedTime:    &currentTime,
		LastTransitionTime: currentTime,
	}
}
