package toolchaincluster

import (
	"context"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// NewReconciler returns a new Reconciler
func NewReconciler(mgr manager.Manager, log logr.Logger, namespace string, timeout time.Duration) *Reconciler {
	cacheLog := log.WithName("toolchaincluster_cache")
	clusterCacheService := cluster.NewToolchainClusterService(mgr.GetClient(), cacheLog, namespace, timeout)
	return &Reconciler{
		client:              mgr.GetClient(),
		scheme:              mgr.GetScheme(),
		log:                 log,
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
	return c.Watch(&source.Kind{Type: &toolchainv1alpha1.ToolchainCluster{}}, &handler.EnqueueRequestForObject{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr manager.Manager) error {
	return add(mgr, r)
}

// Reconciler reconciles a ToolchainCluster object
type Reconciler struct {
	client              client.Client
	scheme              *runtime.Scheme
	log                 logr.Logger
	clusterCacheService cluster.ToolchainClusterService
}

// Reconcile reads that state of the cluster for a ToolchainCluster object and makes changes based on the state read
// and what is in the ToolchainCluster.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
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
