package toolchaincluster_resources

import (
	"context"
	"embed"
	"fmt"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	commoncontroller "github.com/codeready-toolchain/toolchain-common/controllers"
	applycl "github.com/codeready-toolchain/toolchain-common/pkg/client"
	commonpredicates "github.com/codeready-toolchain/toolchain-common/pkg/predicate"
	"github.com/codeready-toolchain/toolchain-common/pkg/template"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, operatorNamespace string) error {
	build := ctrl.NewControllerManagedBy(mgr).
		For(&v1.ServiceAccount{})

	// add watcher for all kinds from given templates
	allObjects, err := template.LoadObjectsFromEmbedFS(r.templates, &template.Variables{Namespace: operatorNamespace})
	if err != nil {
		return err
	}
	mapToOwnerByLabel := handler.EnqueueRequestsFromMapFunc(commoncontroller.MapToOwnerByLabel("", toolchainv1alpha1.ProviderLabelKey))
	for _, obj := range allObjects {
		build = build.Watches(&source.Kind{Type: obj.DeepCopyObject().(runtimeclient.Object)}, mapToOwnerByLabel, builder.WithPredicates(commonpredicates.LabelsAndGenerationPredicate{}))
	}
	return build.Complete(r)
}

// Reconciler reconciles a ToolchainCluster object
type Reconciler struct {
	client    runtimeclient.Client
	scheme    *runtime.Scheme
	templates *embed.FS
}

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
