package template

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/rand"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	templatev1 "github.com/openshift/api/template/v1"
	"github.com/openshift/library-go/pkg/template/generator"
	"github.com/openshift/library-go/pkg/template/templateprocessing"
	"github.com/pkg/errors"
	errs "github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("template_processor")

const LastAppliedConfigurationAnnotationKey = "toolchain.dev.openshift.com/last-applied-configuration"

// Processor the tool that will process and apply a template with variables
type Processor struct {
	cl     client.Client
	scheme *runtime.Scheme
}

// NewProcessor returns a new Processor
func NewProcessor(cl client.Client, scheme *runtime.Scheme) Processor {
	return Processor{cl: cl, scheme: scheme}
}

// Process processes the template (ie, replaces the variables with their actual values) and optionally filters the result
// to return a subset of the template objects
func (p Processor) Process(tmpl *templatev1.Template, values map[string]string, filters ...FilterFunc) ([]runtime.RawExtension, error) {
	// inject variables in the twmplate
	for param, val := range values {
		v := templateprocessing.GetParameterByName(tmpl, param)
		if v != nil {
			v.Value = val
			v.Generate = ""
		}
	}
	// convert the template into a set of objects
	tmplProcessor := templateprocessing.NewProcessor(map[string]generator.Generator{
		"expression": generator.NewExpressionValueGenerator(rand.New(rand.NewSource(time.Now().UnixNano()))),
	})
	if err := tmplProcessor.Process(tmpl); len(err) > 0 {
		return nil, errs.Wrap(err.ToAggregate(), "unable to process template")
	}
	var result templatev1.Template
	if err := p.scheme.Convert(tmpl, &result, nil); err != nil {
		return nil, errs.Wrap(err, "failed to convert template to external template object")
	}
	return Filter(result.Objects, filters...), nil
}

// Apply applies the objects, ie, creates or updates them on the cluster
func (p Processor) Apply(objs []runtime.RawExtension) error {
	for _, rawObj := range objs {
		obj := rawObj.Object
		if obj == nil {
			continue
		}
		gvk := obj.GetObjectKind().GroupVersionKind()
		_, err := p.ApplySingle(obj, true, nil)
		if err != nil {
			return errs.Wrapf(err, "unable to create resource of kind: %s, version: %s", gvk.Kind, gvk.Version)
		}
	}
	return nil
}

// ApplySingle creates the object if is missing and if the owner object is provided, then it's set as a controller reference.
// If the objects exists then based on the UpdateStrategy it's updated or let as it is and updated is skipped.
// The return boolean says if the object was either created or updated
func (p Processor) ApplySingle(obj runtime.Object, forceUpdate bool, owner v1.Object) (bool, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	createdOrUpdated, err := p.createOrUpdateObj(obj, forceUpdate, owner)
	if err != nil {
		return createdOrUpdated, errs.Wrapf(err, "unable to create resource of kind: %s, version: %s", gvk.Kind, gvk.Version)
	}
	return createdOrUpdated, nil
}

func (p Processor) createOrUpdateObj(newResource runtime.Object, forceUpdate bool, owner v1.Object) (bool, error) {
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
		creationErr := p.createObj(newResource, metaNew, owner)
		if apierrors.IsNotFound(err) || creationErr == nil {
			return true, creationErr
		}
		return false, errors.Wrapf(creationErr, "unable to get the resource '%v' and creation failed too: %s", existing, creationErr.Error())
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
	return originalGeneration == metaNewAfterUpdate.GetGeneration(), nil
}

func getNewConfiguration(newResource runtime.Object) string {
	newJson := marshalObjectContent(newResource)

	if len(newJson) == 0 {
		return fmt.Sprintf("%v", newResource)
	}
	return string(newJson)
}

func marshalObjectContent(newResource runtime.Object) []byte {
	var newJson []byte
	var err error
	if newRes, ok := newResource.(runtime.Unstructured); ok {
		newJson, err = json.Marshal(newRes.UnstructuredContent())
		if err != nil {
			log.Error(err, "unable to marshal the object", "object", newRes.UnstructuredContent())
		}
	} else {
		newJson, err = json.Marshal(newResource)
		if err != nil {
			log.Error(err, "unable to marshal the object", "object", newResource)
		}
	}
	return newJson
}

func (p Processor) createObj(newResource runtime.Object, metaNew v1.Object, owner v1.Object) error {
	if owner != nil {
		err := controllerutil.SetControllerReference(owner, metaNew, p.scheme)
		if err != nil {
			return errs.Wrap(err, "unable to set controller references")
		}
	}
	return p.cl.Create(context.TODO(), newResource)
}
