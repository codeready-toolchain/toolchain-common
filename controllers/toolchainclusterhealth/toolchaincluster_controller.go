package toolchainclusterhealth

import (
	"context"
	"fmt"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NewReconciler returns a new Reconciler
func NewReconciler(cl client.Client, scheme runtime.Scheme, requeAfter time.Duration) *Reconciler {
	log.Log.WithName("toolchaincluster_health")
	return &Reconciler{
		client:     cl,
		scheme:     &scheme,
		requeAfter: requeAfter,
	}
}

// Reconciler reconciles a ToolchainCluster object
type Reconciler struct {
	client     client.Client
	scheme     *runtime.Scheme
	requeAfter time.Duration
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
	healthChecker := &HealthChecker{
		localClusterClient:     r.client,
		remoteClusterClient:    cachedCluster.Client,
		remoteClusterClientset: clientSet,
		logger:                 reqLogger,
	}
	//update the status of the individual cluster.
	if err := healthChecker.updateIndividualClusterStatus(ctx, toolchainCluster); err != nil {
		reqLogger.Error(err, "unable to update cluster status of ToolchainCluster")
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: r.requeAfter}, nil
}
