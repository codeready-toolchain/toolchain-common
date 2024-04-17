package toolchainclusterhealth

import (
	"context"
	"fmt"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// defaultHealthCheckAndUpdateClusterStatus updates the status of the individual cluster
func defaultHealthCheckAndUpdateClusterStatus(ctx context.Context, localClusterClient client.Client, remoteClusterClient client.Client, remoteClusterClientset *kubeclientset.Clientset, logger logr.Logger, toolchainCluster *toolchainv1alpha1.ToolchainCluster) error {
	healthChecker := &HealthChecker{
		localClusterClient:     localClusterClient,
		remoteClusterClient:    remoteClusterClient,
		remoteClusterClientset: remoteClusterClientset,
		logger:                 logger,
	}

	// update the status of the individual cluster.
	return healthChecker.updateIndividualClusterStatus(ctx, toolchainCluster)
}

// NewReconciler returns a new Reconciler
func NewReconciler(mgr manager.Manager, namespace string, timeout time.Duration, requeAfter time.Duration) *Reconciler {
	log.Log.WithName("toolchaincluster_health")
	return &Reconciler{
		client:                            mgr.GetClient(),
		scheme:                            mgr.GetScheme(),
		requeAfter:                        requeAfter,
		healthCheckAndUpdateClusterStatus: defaultHealthCheckAndUpdateClusterStatus,
	}
}

// Reconciler reconciles a ToolchainCluster object
type Reconciler struct {
	client     client.Client
	scheme     *runtime.Scheme
	requeAfter time.Duration

	healthCheckAndUpdateClusterStatus func(ctx context.Context, localClusterClient client.Client, remoteClusterClient client.Client, remoteClusterClientset *kubeclientset.Clientset, logger logr.Logger, toolchainCluster *toolchainv1alpha1.ToolchainCluster) error
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
	reqLogger := log.FromContext(ctx).WithName("health")
	reqLogger.Info("Reconciling ToolchainCluster")

	// Fetch the ToolchainCluster instance
	toolchainCluster := &toolchainv1alpha1.ToolchainCluster{}
	err := r.client.Get(ctx, request.NamespacedName, toolchainCluster)
	if err != nil {
		if errors.IsNotFound(err) {
			// Stop monitoring the toolchain cluster as it is deleted
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	cachedCluster, ok := cluster.GetCachedToolchainCluster(toolchainCluster.Name)
	if !ok {
		err := fmt.Errorf("cluster %s not found in cache", toolchainCluster.Name)
		toolchainCluster.Status.Conditions = []toolchainv1alpha1.ToolchainClusterCondition{clusterOfflineCondition()}
		if err := r.client.Status().Update(ctx, toolchainCluster); err != nil {
			reqLogger.Error(err, "failed to update the status of ToolchainCluster")
		}
		return reconcile.Result{}, err
	}

	clientSet, err := kubeclientset.NewForConfig(cachedCluster.RestConfig)
	if err != nil {
		return reconcile.Result{}, err
	}

	//update the status of the individual cluster.
	if err := defaultHealthCheckAndUpdateClusterStatus(ctx, r.client, cachedCluster.Client, clientSet, reqLogger, toolchainCluster); err != nil {
		reqLogger.Error(err, "unable to update cluster status of ToolchainCluster")
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: r.requeAfter}, nil
}
