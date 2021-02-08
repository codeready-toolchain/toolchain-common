package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const LastAppliedConfigurationAnnotationKey = "toolchain.dev.openshift.com/last-applied-configuration"

var log = logf.Log.WithName("apply_client")

type ApplyClient struct {
	cl     client.Client
	scheme *runtime.Scheme
}

// NewApplyClient returns a new ApplyClient
func NewApplyClient(cl client.Client, scheme *runtime.Scheme) *ApplyClient {
	return &ApplyClient{cl: cl, scheme: scheme}
}

type createOrUpdateObjectConfiguration struct {
	owner             v1.Object
	forceUpdate       bool
	saveConfiguration bool
}

func newCreateOrUpdateObjectConfiguration(options ...CreateOrUpdateObjectOption) createOrUpdateObjectConfiguration {
	config := createOrUpdateObjectConfiguration{
		owner:             nil,
		forceUpdate:       false,
		saveConfiguration: true,
	}
	for _, apply := range options {
		apply(&config)
	}
	return config
}

// CreateOrUpdateObjectOption an option when creating or updating a resource
type CreateOrUpdateObjectOption func(*createOrUpdateObjectConfiguration)

// SetOwner sets the owner of the resource (default: `nil`)
func SetOwner(owner v1.Object) CreateOrUpdateObjectOption {
	return func(config *createOrUpdateObjectConfiguration) {
		config.owner = owner
	}
}

// ForceUpdate forces the update of the resource (default: `false`)
func ForceUpdate(forceUpdate bool) CreateOrUpdateObjectOption {
	return func(config *createOrUpdateObjectConfiguration) {
		config.forceUpdate = forceUpdate
	}
}

// SaveConfiguration saves the applied configuration
// in the resource annotations (default: `true`)
func SaveConfiguration(saveConfiguration bool) CreateOrUpdateObjectOption {
	return func(config *createOrUpdateObjectConfiguration) {
		config.saveConfiguration = saveConfiguration
	}
}

// CreateOrUpdateObject creates the object if is missing and if the owner object is provided, then it's set as a controller reference.
// If the objects exists then when the spec content has changed (based on the content of the annotation in the original object) then it
// is automatically updated. If it looks to be same then based on the value of forceUpdate param it updates the object or not.
// The return boolean says if the object was either created or updated (`true`). If nothing changed (ie, the generation was not
// incremented by the server), then it returns `false`.
func (p ApplyClient) CreateOrUpdateObject(obj runtime.Object, options ...CreateOrUpdateObjectOption) (bool, error) {
	// gets the meta accessor to the new resource
	metaNew, err := meta.Accessor(obj)
	if err != nil {
		return false, errors.Wrapf(err, "cannot get metadata from %+v", obj)
	}
	config := newCreateOrUpdateObjectConfiguration(options...)

	// creates a deepcopy of the new resource to be used to check if it already exists
	existing := obj.DeepCopyObject()

	var newConfiguration string
	if config.saveConfiguration {
		// set current object as annotation
		annotations := metaNew.GetAnnotations()
		newConfiguration = getNewConfiguration(obj)
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[LastAppliedConfigurationAnnotationKey] = newConfiguration
		metaNew.SetAnnotations(annotations)
	}
	// gets current object (if exists)
	namespacedName := types.NamespacedName{Namespace: metaNew.GetNamespace(), Name: metaNew.GetName()}
	if err := p.cl.Get(context.TODO(), namespacedName, existing); err != nil {
		if apierrors.IsNotFound(err) {
			return true, p.createObj(obj, metaNew, config.owner)
		}
		return false, errors.Wrapf(err, "unable to get the resource '%v'", existing)
	}

	// gets the meta accessor to the existing resource
	metaExisting, err := meta.Accessor(existing)
	if err != nil {
		return false, errors.Wrapf(err, "cannot get metadata from %+v", existing)
	}

	// as it already exists, check using the UpdateStrategy if it should be updated
	if !config.forceUpdate {
		existingAnnotations := metaExisting.GetAnnotations()
		if existingAnnotations != nil {
			if newConfiguration == existingAnnotations[LastAppliedConfigurationAnnotationKey] {
				return false, nil
			}
		}
	}

	// retrieve the current 'resourceVersion' to set it in the resource passed to the `client.Update()`
	// otherwise we would get an error with the following message:
	// `nstemplatetiers.toolchain.dev.openshift.com "basic" is invalid: metadata.resourceVersion: Invalid value: 0x0: must be specified for an update`
	originalGeneration := metaExisting.GetGeneration()
	metaNew.SetResourceVersion(metaExisting.GetResourceVersion())

	// also, if the resource to create is a Service and there's a previous version, we should retain its `spec.ClusterIP`, otherwise
	// the update will fail with the following error:
	// `Service "<name>" is invalid: spec.clusterIP: Invalid value: "": field is immutable`
	if err := RetainClusterIP(obj, existing); err != nil {
		return false, err
	}
	if err := p.cl.Update(context.TODO(), obj); err != nil {
		return false, errors.Wrapf(err, "unable to update the resource '%v'", obj)
	}

	// gets the meta accessor to the resource that was updated
	metaNewAfterUpdate, err := meta.Accessor(obj)
	if err != nil {
		return false, errors.Wrapf(err, "cannot get metadata from %+v", obj)
	}

	// check if it was changed or not
	return originalGeneration != metaNewAfterUpdate.GetGeneration(), nil
}

// RetainClusterIP sets the `spec.clusterIP` value from the given 'existing' object
// into the 'newResource' object.
func RetainClusterIP(newResource, existing runtime.Object) error {
	clusterIP, found, err := clusterIP(existing)
	if err != nil {
		return err
	}
	if !found {
		// skip
		return nil
	}
	switch newResource := newResource.(type) {
	case *corev1.Service:
		newResource.Spec.ClusterIP = clusterIP
	case *unstructured.Unstructured:
		if err := unstructured.SetNestedField(newResource.Object, clusterIP, "spec", "clusterIP"); err != nil {
			return err
		}
	default:
		// do nothing, object is not a service
	}
	return nil
}

func clusterIP(obj runtime.Object) (string, bool, error) {
	switch obj := obj.(type) {
	case *corev1.Service:
		return obj.Spec.ClusterIP, obj.Spec.ClusterIP != "", nil
	case *unstructured.Unstructured:
		return unstructured.NestedString(obj.Object, "spec", "clusterIP")
	default:
		// do nothing, object is not a service
		return "", false, nil
	}
}

func getNewConfiguration(newResource runtime.Object) string {
	newJSON, err := marshalObjectContent(newResource)
	if err != nil {
		log.Error(err, "unable to marshal the object", "object", newResource)
		return fmt.Sprintf("%v", newResource)
	}
	return string(newJSON)
}

func marshalObjectContent(newResource runtime.Object) ([]byte, error) {
	if newRes, ok := newResource.(runtime.Unstructured); ok {
		return json.Marshal(newRes.UnstructuredContent())
	}
	return json.Marshal(newResource)
}

func (p ApplyClient) createObj(newResource runtime.Object, metaNew v1.Object, owner v1.Object) error {
	if owner != nil {
		err := controllerutil.SetControllerReference(owner, metaNew, p.scheme)
		if err != nil {
			return errors.Wrap(err, "unable to set controller references")
		}
	}
	return p.cl.Create(context.TODO(), newResource)
}

// Apply applies the objects, ie, creates or updates them on the cluster
// returns `true, nil` if at least one of the objects was created or modified,
// `false, nil` if nothing changed, and `false, err` if an error occurred
func (p ApplyClient) Apply(toolchainObjects []ToolchainObject, newLabels map[string]string) (bool, error) {
	createdOrUpdated := false
	for _, toolchainObject := range toolchainObjects {
		// set newLabels
		labels := toolchainObject.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		for key, value := range newLabels {
			labels[key] = value
		}
		toolchainObject.SetLabels(labels)

		gvk := toolchainObject.GetGvk()
		result, err := p.CreateOrUpdateObject(toolchainObject.GetRuntimeObject(), ForceUpdate(true))
		if err != nil {
			return false, errors.Wrapf(err, "unable to create resource of kind: %s, version: %s", gvk.Kind, gvk.Version)
		}
		createdOrUpdated = createdOrUpdated || result
	}
	return createdOrUpdated, nil
}
