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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ResourceControllerLabelValue is being added to all the resources managed by this controller.
// It's then used to filter all the events on those resources by using a mapper function in the watcher configuration.
const ResourceControllerLabelValue = "toolchaincluster-resources-controller" // todo move this label value to api repo

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, operatorNamespace string) error {
	// check for required templates FS directory
	if r.templates == nil {
		return fmt.Errorf("no templates FS configured")
	}

	build := ctrl.NewControllerManagedBy(mgr).
		For(&v1.ServiceAccount{})

	// add watcher for all kinds from given templates
	var err error
	r.templateObjects, err = template.LoadObjectsFromEmbedFS(r.templates, &template.Variables{Namespace: operatorNamespace})
	if err != nil {
		return err
	}
	mapToOwnerByLabel := handler.EnqueueRequestsFromMapFunc(commoncontroller.MapToControllerByMatchingLabel(operatorNamespace, toolchainv1alpha1.ProviderLabelKey, ResourceControllerLabelValue))
	for _, obj := range r.templateObjects {
		build = build.Watches(&source.Kind{Type: obj.DeepCopyObject().(runtimeclient.Object)}, mapToOwnerByLabel, builder.WithPredicates(commonpredicates.LabelsAndGenerationPredicate{}))
	}
	return build.Complete(r)
}

// Reconciler reconciles a ToolchainCluster object
type Reconciler struct {
	client          runtimeclient.Client
	scheme          *runtime.Scheme
	templates       *embed.FS
	templateObjects []*unstructured.Unstructured
}

// Reconcile loads all the manifests from a given embed.FS folder, evaluates the supported variables and applies the objects in the cluster.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling ToolchainCluster resources controller")
	// check for required templates FS directory
	if r.templates == nil {
		return reconcile.Result{}, fmt.Errorf("no templates FS configured")
	}

	// apply all the objects with a custom label
	newLabels := map[string]string{
		toolchainv1alpha1.ProviderLabelKey: ResourceControllerLabelValue,
	}

	// todo implement delete logic for objects that were renamed/removed from the templates

	return reconcile.Result{}, applycl.ApplyUnstructuredObjects(ctx, r.client, r.templateObjects, newLabels) // apply objects on the cluster
}
