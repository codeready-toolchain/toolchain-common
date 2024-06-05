package toolchaincluster

import (
	"context"
	"fmt"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a ToolchainCluster object
type Reconciler struct {
	Client                  client.Client
	Scheme                  *runtime.Scheme
	RequeAfter              time.Duration
	overwriteRunHealthcheck func(ctx context.Context, lcl client.Client, rcl client.Client, rclset *kubeclientset.Clientset, tc *toolchainv1alpha1.ToolchainCluster, lgr logr.Logger) ([]toolchainv1alpha1.Condition, error)
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

	// this is a migration step to make sure that we are forwards-compatible with
	// the secrets labeled by the toolchainCluster name, which are going to be the basis
	// for *creating* toolchain clusters in the future.
	if err := r.labelTokenSecret(ctx, toolchainCluster); err != nil {
		reqLogger.Error(err, "unable to check the labels in the associated secret")
		return reconcile.Result{}, err
	}

	cachedCluster, ok := cluster.GetCachedToolchainCluster(toolchainCluster.Name)
	if !ok {
		err := fmt.Errorf("cluster %s not found in cache", toolchainCluster.Name)
		toolchainCluster.Status.Conditions = []toolchainv1alpha1.Condition{clusterOfflineCondition()}
		if err := r.Client.Status().Update(ctx, toolchainCluster); err != nil {
			reqLogger.Error(err, "failed to update the status of ToolchainCluster")
		}
		return reconcile.Result{}, err
	}

	clientSet, err := kubeclientset.NewForConfig(cachedCluster.RestConfig)
	if err != nil {
		reqLogger.Error(err, "cannot create ClientSet for the ToolchainCluster")
		return reconcile.Result{}, err
	}

	// execute healthcheck
	healthcheckResult, err := r.runHealthcheck(ctx, r.Client, cachedCluster.Client, clientSet, toolchainCluster, reqLogger)
	if err != nil {
		reqLogger.Error(err, "unable to run healthcheck for ToolchainCluster")
		return reconcile.Result{}, err
	}
	// update the status of the individual cluster.
	if err := r.updateStatus(ctx, toolchainCluster, healthcheckResult); err != nil {
		reqLogger.Error(err, "unable to update cluster status of ToolchainCluster")
		return reconcile.Result{}, err
	}
	return reconcile.Result{RequeueAfter: r.RequeAfter}, nil
}
func (r *Reconciler) runHealthcheck(ctx context.Context, lcl client.Client, rcl client.Client, rclset *kubeclientset.Clientset, tc *toolchainv1alpha1.ToolchainCluster, lgr logr.Logger) ([]toolchainv1alpha1.Condition, error) {
	if r.overwriteRunHealthcheck != nil {
		return r.overwriteRunHealthcheck(ctx, lcl, rcl, rclset, tc, lgr)
	}
	healthChecker := &HealthChecker{
		localClusterClient:     lcl,
		remoteClusterClient:    rcl,
		remoteClusterClientset: rclset,
		logger:                 lgr,
	}
	// update the status of the individual cluster.
	clstatus := healthChecker.getClusterHealthStatus(ctx)
	return clstatus, nil

}

func (r *Reconciler) updateStatus(ctx context.Context, toolchainCluster *toolchainv1alpha1.ToolchainCluster, currentconditions []toolchainv1alpha1.Condition) error {

	for index, currentCond := range currentconditions {
		for _, previousCond := range toolchainCluster.Status.Conditions {
			if currentCond.Type == previousCond.Type && currentCond.Status == previousCond.Status {
				currentconditions[index].LastTransitionTime = previousCond.LastTransitionTime
			}
		}
	}

	toolchainCluster.Status.Conditions = currentconditions
	if err := r.Client.Status().Update(ctx, toolchainCluster); err != nil {
		return errors.Wrapf(err, "Failed to update the status of cluster %s", toolchainCluster.Name)
	}
	return nil
}

func (r *Reconciler) labelTokenSecret(ctx context.Context, toolchainCluster *toolchainv1alpha1.ToolchainCluster) error {
	if toolchainCluster.Spec.SecretRef.Name == "" {
		return nil
	}

	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: toolchainCluster.Spec.SecretRef.Name, Namespace: toolchainCluster.Namespace}, secret); err != nil {
		if kerrors.IsNotFound(err) {
			// The referenced secret does not exist yet, so we can't really label it.
			// Because the reconciler runs periodically (not just on ToolchainCluster change), we will
			// recover from this condition once the secret appears in the cluster.
			log.FromContext(ctx).Info("failed to find the referenced secret. Cluster cache might be broken until it is created.", "expectedSecretName", toolchainCluster.Spec.SecretRef.Name)
			return nil
		}
		return err
	}

	if secret.Labels[toolchainv1alpha1.ToolchainClusterLabel] != toolchainCluster.Name {
		if secret.Labels == nil {
			secret.Labels = map[string]string{}
		}

		secret.Labels[toolchainv1alpha1.ToolchainClusterLabel] = toolchainCluster.Name

		if err := r.Client.Update(ctx, secret); err != nil {
			return err
		}
	}

	return nil
}
