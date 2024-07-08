package toolchaincluster

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/codeready-toolchain/api/api/v1alpha1"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a ToolchainCluster object
type Reconciler struct {
	Client      client.Client
	Scheme      *runtime.Scheme
	RequeAfter  time.Duration
	checkHealth func(context.Context, *kubeclientset.Clientset) (bool, error)
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&toolchainv1alpha1.ToolchainCluster{}).
		Complete(r)
}

// Reconcile reads that state of the cluster for a ToolchainCluster object and makes changes based on the state read
// and what is in the ToolchainCluster.Spec. It updates the status of the individual cluster
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling ToolchainCluster")

	// Fetch the ToolchainCluster instance
	toolchainCluster := &toolchainv1alpha1.ToolchainCluster{}
	err := r.Client.Get(ctx, request.NamespacedName, toolchainCluster)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Stop monitoring the toolchain cluster as it is deleted
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	cachedCluster, ok := cluster.GetCachedToolchainCluster(toolchainCluster.Name)
	if !ok {
		err := fmt.Errorf("cluster %s not found in cache", toolchainCluster.Name)
		if err := r.updateStatus(ctx, toolchainCluster, clusterNotReadyCondition()); err != nil {
			reqLogger.Error(err, "unable to update cluster status of ToolchainCluster")
		}
		return reconcile.Result{}, err
	}

	if err = r.migrateSecretToKubeConfig(ctx, toolchainCluster); err != nil {
		return reconcile.Result{}, err
	}

	clientSet, err := kubeclientset.NewForConfig(cachedCluster.RestConfig)
	if err != nil {
		reqLogger.Error(err, "cannot create ClientSet for the ToolchainCluster")
		if err := r.updateStatus(ctx, toolchainCluster, clusterNotReadyCondition()); err != nil {
			reqLogger.Error(err, "unable to update cluster status of ToolchainCluster")
		}
		return reconcile.Result{}, err
	}

	// execute healthcheck
	healthCheckResult := r.getClusterHealthCondition(ctx, clientSet)

	// update the status of the individual cluster.
	if err := r.updateStatus(ctx, toolchainCluster, healthCheckResult); err != nil {
		reqLogger.Error(err, "unable to update cluster status of ToolchainCluster")
		return reconcile.Result{}, err
	}
	return reconcile.Result{RequeueAfter: r.RequeAfter}, nil
}

func (r *Reconciler) updateStatus(ctx context.Context, toolchainCluster *toolchainv1alpha1.ToolchainCluster, currentconditions ...toolchainv1alpha1.Condition) error {
	toolchainCluster.Status.Conditions = condition.AddOrUpdateStatusConditionsWithLastUpdatedTimestamp(toolchainCluster.Status.Conditions, currentconditions...)
	if err := r.Client.Status().Update(ctx, toolchainCluster); err != nil {
		return fmt.Errorf("failed to update the status of cluster - %s "+err.Error(), toolchainCluster.Name)
	}
	return nil
}

func (r *Reconciler) getClusterHealthCondition(ctx context.Context, remoteClusterClientset *kubeclientset.Clientset) v1alpha1.Condition {

	ishealthy, err := r.getClusterHealth(ctx, remoteClusterClientset)
	if err != nil {
		return clusterOfflineCondition(err.Error())
	} else {
		if !ishealthy {
			return clusterNotReadyCondition()
		}
		return clusterReadyCondition()
	}
}

func (r *Reconciler) getClusterHealth(ctx context.Context, remoteClusterClientset *kubeclientset.Clientset) (bool, error) {
	if r.checkHealth != nil {
		return r.checkHealth(ctx, remoteClusterClientset)
	}
	return getClusterHealthStatus(ctx, remoteClusterClientset)
}
func clusterOfflineCondition(errMsg string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  toolchainv1alpha1.ToolchainClusterClusterNotReachableReason,
		Message: errMsg,
	}
}

func clusterReadyCondition() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionTrue,
		Reason:  toolchainv1alpha1.ToolchainClusterClusterReadyReason,
		Message: healthzOk,
	}
}

func clusterNotReadyCondition() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  toolchainv1alpha1.ToolchainClusterClusterNotReadyReason,
		Message: healthzNotOk,
	}
}

func (r *Reconciler) migrateSecretToKubeConfig(ctx context.Context, tc *toolchainv1alpha1.ToolchainCluster) error {
	if len(tc.Spec.SecretRef.Name) == 0 {
		return nil
	}

	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: tc.Spec.SecretRef.Name, Namespace: tc.Namespace}, secret); err != nil {
		return err
	}

	token := secret.Data["token"]
	apiEndpoint := tc.Spec.APIEndpoint
	operatorNamespace := tc.Labels["namespace"]
	insecureTls := len(tc.Spec.DisabledTLSValidations) == 1 && tc.Spec.DisabledTLSValidations[0] == "*"
	// we ignore the Spec.CABundle here because we don't want it migrated. The new configurations are free
	// to use the certificate data for the connections but we don't want to migrate the existing certificates.
	kubeConfig := composeKubeConfigFromData(token, apiEndpoint, operatorNamespace, insecureTls)

	data, err := clientcmd.Write(kubeConfig)
	if err != nil {
		return err
	}

	origKubeConfigData := secret.Data["kubeconfig"]
	secret.Data["kubeconfig"] = data

	if !bytes.Equal(origKubeConfigData, data) {
		if err = r.Client.Update(ctx, secret); err != nil {
			return err
		}
	}

	return nil
}

func composeKubeConfigFromData(token []byte, apiEndpoint, operatorNamespace string, insecureTls bool) clientcmdapi.Config {
	return clientcmdapi.Config{
		Contexts: map[string]*clientcmdapi.Context{
			"ctx": {
				Cluster:   "cluster",
				Namespace: operatorNamespace,
				AuthInfo:  "auth",
			},
		},
		CurrentContext: "ctx",
		Clusters: map[string]*clientcmdapi.Cluster{
			"cluster": {
				Server:                apiEndpoint,
				InsecureSkipTLSVerify: insecureTls,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"auth": {
				Token: string(token),
			},
		},
	}
}
