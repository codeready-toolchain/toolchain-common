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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// LastAppliedConfigurationAnnotationKey the key to save the last applied configuration in the resource annotations
const LastAppliedConfigurationAnnotationKey = "toolchain.dev.openshift.com/last-applied-configuration"

var log = logf.Log.WithName("apply_client")

// ApplyClient the client to use when creating or updating objects
type ApplyClient struct {
	Client client.Client
	scheme *runtime.Scheme
}

// NewApplyClient returns a new ApplyClient
func NewApplyClient(cl client.Client, scheme *runtime.Scheme) *ApplyClient {
	return &ApplyClient{Client: cl, scheme: scheme}
}

type applyObjectConfiguration struct {
	owner             v1.Object
	forceUpdate       bool
	saveConfiguration bool
}

func newApplyObjectConfiguration(options ...ApplyObjectOption) applyObjectConfiguration {
	config := applyObjectConfiguration{
		owner:             nil,
		forceUpdate:       false,
		saveConfiguration: true,
	}
	for _, apply := range options {
		apply(&config)
	}
	return config
}

// ApplyObjectOption an option when creating or updating a resource
type ApplyObjectOption func(*applyObjectConfiguration)

// SetOwner sets the owner of the resource (default: `nil`)
func SetOwner(owner v1.Object) ApplyObjectOption {
	return func(config *applyObjectConfiguration) {
		config.owner = owner
	}
}

// ForceUpdate forces the update of the resource (default: `false`)
func ForceUpdate(forceUpdate bool) ApplyObjectOption {
	return func(config *applyObjectConfiguration) {
		config.forceUpdate = forceUpdate
	}
}

// SaveConfiguration saves the applied configuration
// in the resource annotations (default: `true`)
func SaveConfiguration(saveConfiguration bool) ApplyObjectOption {
	return func(config *applyObjectConfiguration) {
		config.saveConfiguration = saveConfiguration
	}
}

// ApplyRuntimeObject casts the provided object to client.Object and calls ApplyClient.ApplyObject method
func (p ApplyClient) ApplyRuntimeObject(obj runtime.Object, options ...ApplyObjectOption) (bool, error) {
	clientObj, ok := obj.(client.Object)
	if !ok {
		return false, fmt.Errorf("unable to cast of the object to client.Object: %+v", obj)
	}
	return p.applyObject(clientObj, options...)
}

// ApplyObject creates the object if is missing and if the owner object is provided, then it's set as a controller reference.
// If the objects exists then when the spec content has changed (based on the content of the annotation in the original object) then it
// is automatically updated. If it looks to be same then based on the value of forceUpdate param it updates the object or not.
// The return boolean says if the object was either created or updated (`true`). If nothing changed (ie, the generation was not
// incremented by the server), then it returns `false`.
func (p ApplyClient) ApplyObject(obj client.Object, options ...ApplyObjectOption) (bool, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	createdOrUpdated, err := p.applyObject(obj, options...)
	if err != nil {
		return createdOrUpdated, errors.Wrapf(err, "unable to create resource of kind: %s, version: %s", gvk.Kind, gvk.Version)
	}
	return createdOrUpdated, nil
}

func (p ApplyClient) applyObject(obj client.Object, options ...ApplyObjectOption) (bool, error) {
	// gets the meta accessor to the new resource
	// gets the meta accessor to the new resource
	config := newApplyObjectConfiguration(options...)

	// creates a deepcopy of the new resource to be used to check if it already exists
	existing := obj.DeepCopyObject().(client.Object)

	var newConfiguration string
	if config.saveConfiguration {
		// set current object as annotation
		annotations := obj.GetAnnotations()
		newConfiguration = getNewConfiguration(obj)
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[LastAppliedConfigurationAnnotationKey] = newConfiguration
		obj.SetAnnotations(annotations)
	}
	// gets current object (if exists)
	namespacedName := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	if err := p.Client.Get(context.TODO(), namespacedName, existing); err != nil {
		if apierrors.IsNotFound(err) {
			return true, p.createObj(obj, config.owner)
		}
		return false, errors.Wrapf(err, "unable to get the resource '%v'", existing)
	}

	// as it already exists, check using the UpdateStrategy if it should be updated
	if !config.forceUpdate {
		existingAnnotations := existing.GetAnnotations()
		if existingAnnotations != nil {
			if newConfiguration == existingAnnotations[LastAppliedConfigurationAnnotationKey] {
				return false, nil
			}
		}
	}

	// retrieve the current 'resourceVersion' to set it in the resource passed to the `client.Update()`
	// otherwise we would get an error with the following message:
	// `nstemplatetiers.toolchain.dev.openshift.com "basic" is invalid: metadata.resourceVersion: Invalid value: 0x0: must be specified for an update`
	originalGeneration := existing.GetGeneration()
	obj.SetResourceVersion(existing.GetResourceVersion())

	// also, if the resource to create is a Service and there's a previous version, we should retain its `spec.ClusterIP`, otherwise
	// the update will fail with the following error:
	// `Service "<name>" is invalid: spec.clusterIP: Invalid value: "": field is immutable`
	if err := RetainClusterIP(obj, existing); err != nil {
		return false, err
	}
	if err := p.Client.Update(context.TODO(), obj); err != nil {
		return false, errors.Wrapf(err, "unable to update the resource '%v'", obj)
	}

	// check if it was changed or not
	return originalGeneration != obj.GetGeneration(), nil
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

func (p ApplyClient) createObj(newResource client.Object, owner v1.Object) error {
	if owner != nil {
		err := controllerutil.SetControllerReference(owner, newResource, p.scheme)
		if err != nil {
			return errors.Wrap(err, "unable to set controller references")
		}
	}
	return p.Client.Create(context.TODO(), newResource)
}

// ApplyToolchainObjects applies the objects, ie, creates or updates them on the cluster
// returns `true, nil` if at least one of the objects was created or modified,
// `false, nil` if nothing changed, and `false, err` if an error occurred
func (p ApplyClient) ApplyToolchainObjects(toolchainObjects []client.Object, newLabels map[string]string) (bool, error) {
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

		result, err := p.ApplyObject(toolchainObject, ForceUpdate(true))
		if err != nil {
			return false, errors.Wrapf(err, "unable to create resource of kind: %s, version: %s", toolchainObject.GetObjectKind().GroupVersionKind().Kind, toolchainObject.GetObjectKind().GroupVersionKind().Version)
		}
		createdOrUpdated = createdOrUpdated || result
	}
	return createdOrUpdated, nil
}
