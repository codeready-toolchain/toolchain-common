package toolchaincluster_resources

import (
	"context"
	"embed"
	"fmt"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	applycl "github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/template"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&toolchainv1alpha1.ToolchainCluster{}).
		Complete(r)
}

// Reconciler reconciles a ToolchainCluster object
type Reconciler struct {
	client    runtimeclient.Client
	scheme    *runtime.Scheme
	templates *embed.FS
}

//+kubebuilder:rbac:groups=rbac.authorization.k8s.io;authorization.openshift.io,resources=rolebindings;roles;clusterroles;clusterrolebindings,verbs=*
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete

// Reconcile loads all the manifests from a given embed.FS folder, evaluates the supported variables and applies the objects in the cluster.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling ToolchainCluster resources controller")
	// check for required templates FS directory
	if r.templates == nil {
		return reconcile.Result{}, fmt.Errorf("no templates FS configured")
	}

	// Fetch the ToolchainCluster instance
	toolchainCluster := &toolchainv1alpha1.ToolchainCluster{}
	err := r.client.Get(ctx, request.NamespacedName, toolchainCluster)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// read the template fs
	allObjects, err := template.LoadObjectsFromEmbedFS(r.templates, &template.Variables{Namespace: request.Namespace})
	if err != nil {
		return reconcile.Result{}, err
	}
	// apply all the objects and add toolchaincluster as owner reference
	newLabels := map[string]string{
		toolchainv1alpha1.ProviderLabelKey: toolchainv1alpha1.ProviderLabelValue,
		toolchainv1alpha1.OwnerLabelKey:    toolchainCluster.GetName(),
	}
	_, err = applycl.ApplyUnstructuredObjects(ctx, r.client, allObjects, newLabels)
	return reconcile.Result{}, err
}
