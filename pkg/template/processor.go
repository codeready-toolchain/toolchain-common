package template

import (
	"math/rand"
	"time"

	apply "github.com/codeready-toolchain/toolchain-common/pkg/client"
	templatev1 "github.com/openshift/api/template/v1"
	"github.com/openshift/library-go/pkg/template/generator"
	"github.com/openshift/library-go/pkg/template/templateprocessing"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Processor the tool that will process and apply a template with variables
type Processor struct {
	cl     *apply.ApplyClient
	scheme *runtime.Scheme
}

// NewProcessor returns a new Processor
func NewProcessor(cl client.Client, scheme *runtime.Scheme) Processor {
	return Processor{
		cl:     apply.NewApplyClient(cl, scheme),
		scheme: scheme,
	}
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
		return nil, errors.Wrap(err.ToAggregate(), "unable to process template")
	}
	var result templatev1.Template
	if err := p.scheme.Convert(tmpl, &result, nil); err != nil {
		return nil, errors.Wrap(err, "failed to convert template to external template object")
	}
	return Filter(result.Objects, filters...), nil
}

// Apply applies the objects, ie, creates or updates them on the cluster
// returns `true, nil` if at least one of the objects was created or modified,
// `false, nil` if nothing changed, and `false, err` if an error occurred
func (p Processor) Apply(objs []runtime.RawExtension) (bool, error) {
	createdOrUpdated := false
	for _, rawObj := range objs {
		obj := rawObj.Object
		if obj == nil {
			continue
		}
		gvk := obj.GetObjectKind().GroupVersionKind()
		result, err := p.cl.CreateOrUpdateObject(obj, true, nil)
		if err != nil {
			return false, errors.Wrapf(err, "unable to create resource of kind: %s, version: %s", gvk.Kind, gvk.Version)
		}
		createdOrUpdated = createdOrUpdated || result
	}
	return createdOrUpdated, nil
}
