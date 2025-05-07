package toolchainclustercache

import (
	"context"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NewReconciler returns a new Reconciler
func NewReconciler(mgr manager.Manager, namespace string, timeout time.Duration) *Reconciler {
	cacheLog := log.Log.WithName("toolchaincluster_cache")
	clusterCacheService := cluster.NewToolchainClusterService(mgr.GetClient(), cacheLog, namespace, timeout)
	return &Reconciler{
		client:              mgr.GetClient(),
		scheme:              mgr.GetScheme(),
		clusterCacheService: clusterCacheService,
		namespace:           namespace,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("ToolchainClusterCache").
		For(&toolchainv1alpha1.ToolchainCluster{}, builder.WithPredicates(namespacePredicate{namespace: r.namespace})).
		Complete(r)
}

// Reconciler reconciles a ToolchainCluster object
type Reconciler struct {
	client              client.Client
	scheme              *runtime.Scheme
	clusterCacheService cluster.ToolchainClusterService
	namespace           string
}

// Reconcile reads that state of the cluster for a ToolchainCluster object and makes changes based on the state read
// and what is in the ToolchainCluster.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx).WithName("cache")
	reqLogger.Info("Reconciling ToolchainCluster")

	// Fetch the ToolchainCluster instance
	toolchainCluster := &toolchainv1alpha1.ToolchainCluster{}
	err := r.client.Get(ctx, request.NamespacedName, toolchainCluster)
	if err != nil {
		if errors.IsNotFound(err) {
			r.clusterCacheService.DeleteToolchainCluster(request.Name)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, r.clusterCacheService.AddOrUpdateToolchainCluster(toolchainCluster)
}
