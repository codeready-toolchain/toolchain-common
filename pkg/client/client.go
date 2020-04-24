package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// CreateOrUpdateObject creates the object if is missing and if the owner object is provided, then it's set as a controller reference.
// If the objects exists then when the spec content has changed (based on the content of the annotation in the original object) then it
// is automatically updated. If it looks to be same then based on the value of forceUpdate param it updates the object or not.
// The return boolean says if the object was either created or updated (`true`). If nothing changed (ie, the generation was not
// incremented by the server), then it returns `false`.
func (p ApplyClient) CreateOrUpdateObject(obj runtime.Object, forceUpdate bool, owner v1.Object) (bool, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	createdOrUpdated, err := p.createOrUpdateObj(obj, forceUpdate, owner)
	if err != nil {
		return createdOrUpdated, errors.Wrapf(err, "unable to create resource of kind: %s, version: %s", gvk.Kind, gvk.Version)
	}
	return createdOrUpdated, nil
}

func (p ApplyClient) createOrUpdateObj(newResource runtime.Object, forceUpdate bool, owner v1.Object) (bool, error) {
	// gets the meta accessor to the new resource
	metaNew, err := meta.Accessor(newResource)
	if err != nil {
		return false, errors.Wrapf(err, "cannot get metadata from %+v", newResource)
	}

	// creates a deepcopy of the new resource to be used to check if it already exists
	existing := newResource.DeepCopyObject()

	// set current object as annotation
	annotations := metaNew.GetAnnotations()
	newConfiguration := getNewConfiguration(newResource)
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[LastAppliedConfigurationAnnotationKey] = newConfiguration
	metaNew.SetAnnotations(annotations)

	// gets current object (if exists)
	namespacedName := types.NamespacedName{Namespace: metaNew.GetNamespace(), Name: metaNew.GetName()}
	if err := p.cl.Get(context.TODO(), namespacedName, existing); err != nil {
		if apierrors.IsNotFound(err) {
			return true, p.createObj(newResource, metaNew, owner)
		}
		return false, errors.Wrapf(err, "unable to get the resource '%v'", existing)
	}

	// gets the meta accessor to the existing resource
	metaExisting, err := meta.Accessor(existing)
	if err != nil {
		return false, errors.Wrapf(err, "cannot get metadata from %+v", existing)
	}

	// as it already exists, check using the UpdateStrategy if it should be updated
	if !forceUpdate {
		existingAnnotations := metaExisting.GetAnnotations()
		if existingAnnotations != nil {
			if newConfiguration == existingAnnotations[LastAppliedConfigurationAnnotationKey] {
				return false, nil
			}
		}
	}

	// retrieve the current 'resourceVersion' to set it in the resource passed to the `client.Update()`
	// otherwise we would get an error with the following message:
	// "nstemplatetiers.toolchain.dev.openshift.com \"basic\" is invalid: metadata.resourceVersion: Invalid value: 0x0: must be specified for an update"
	originalGeneration := metaExisting.GetGeneration()
	metaNew.SetResourceVersion(metaExisting.GetResourceVersion())
	if err := p.cl.Update(context.TODO(), newResource); err != nil {
		return false, errors.Wrapf(err, "unable to update the resource '%v'", newResource)
	}

	// gets the meta accessor to the resource that was updated
	metaNewAfterUpdate, err := meta.Accessor(newResource)
	if err != nil {
		return false, errors.Wrapf(err, "cannot get metadata from %+v", newResource)
	}

	// check if it was changed or not
	return originalGeneration != metaNewAfterUpdate.GetGeneration(), nil
}

func getNewConfiguration(newResource runtime.Object) string {
	newJson, err := marshalObjectContent(newResource)
	if err != nil {
		log.Error(err, "unable to marshal the object", "object", newResource)
		return fmt.Sprintf("%v", newResource)
	}
	return string(newJson)
}

func marshalObjectContent(newResource runtime.Object) ([]byte, error) {
	if newRes, ok := newResource.(runtime.Unstructured); ok {
		return json.Marshal(newRes.UnstructuredContent())
	} else {
		return json.Marshal(newResource)
	}
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
func (p ApplyClient) Apply(objs []runtime.RawExtension, newLabels map[string]string) (bool, error) {
	createdOrUpdated := false
	for _, rawObj := range objs {
		obj := rawObj.Object
		if obj == nil {
			continue
		}

		acc, err := meta.Accessor(obj)
		if err != nil {
			return false, errors.Wrapf(err, "unable to get the accessor interface of the object '%v'", rawObj.Object)
		}

		// set newLabels
		labels := acc.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		for key, value := range newLabels {
			labels[key] = value
		}
		acc.SetLabels(labels)

		gvk := obj.GetObjectKind().GroupVersionKind()
		result, err := p.CreateOrUpdateObject(obj, true, nil)
		if err != nil {
			return false, errors.Wrapf(err, "unable to create resource of kind: %s, version: %s", gvk.Kind, gvk.Version)
		}
		createdOrUpdated = createdOrUpdated || result
	}
	return createdOrUpdated, nil
}
