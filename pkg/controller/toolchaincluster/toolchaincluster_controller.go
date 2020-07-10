package toolchaincluster

import (
	"context"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_toolchaincluster")

// Add creates a new ToolchainCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, timeout time.Duration) error {
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}
	return add(mgr, newReconciler(mgr, namespace, timeout))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, namespace string, timeout time.Duration) reconcile.Reconciler {
	logger := logf.Log.WithName("toolchaincluster_cache")
	clusterCacheService := cluster.NewToolchainClusterService(mgr.GetClient(), logger, namespace, timeout)
	return &ReconcileToolchainCluster{
		client:              mgr.GetClient(),
		scheme:              mgr.GetScheme(),
		clusterCacheService: clusterCacheService,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("toolchaincluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ToolchainCluster
	return c.Watch(&source.Kind{Type: &toolchainv1alpha1.ToolchainCluster{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
}

// blank assignment to verify that ReconcileToolchainCluster implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileToolchainCluster{}

// ReconcileToolchainCluster reconciles a ToolchainCluster object
type ReconcileToolchainCluster struct {
	client              client.Client
	scheme              *runtime.Scheme
	clusterCacheService cluster.ToolchainClusterService
}

// Reconcile reads that state of the cluster for a ToolchainCluster object and makes changes based on the state read
// and what is in the ToolchainCluster.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileToolchainCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ToolchainCluster")

	// Fetch the ToolchainCluster instance
	toolchainCluster := &toolchainv1alpha1.ToolchainCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, toolchainCluster)
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
