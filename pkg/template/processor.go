package template

import (
	"math/rand"
	"time"

	"github.com/codeready-toolchain/toolchain-common/pkg/client"
	templatev1 "github.com/openshift/api/template/v1"
	"github.com/openshift/library-go/pkg/template/generator"
	"github.com/openshift/library-go/pkg/template/templateprocessing"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

// Processor the tool that will process and apply a template with variables
type Processor struct {
	scheme *runtime.Scheme
}

// NewProcessor returns a new Processor
func NewProcessor(scheme *runtime.Scheme) Processor {
	return Processor{
		scheme: scheme,
	}
}

// Process processes the template (ie, replaces the variables with their actual values) and optionally filters the result
// to return a subset of the template objects
func (p Processor) Process(tmpl *templatev1.Template, values map[string]string, filters ...FilterFunc) ([]client.ToolchainObject, error) {
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
	filtered := Filter(result.Objects, filters...)
	objects := make([]client.ToolchainObject, len(filtered))
	for i, rawObject := range filtered {
		toolchainObject, err := client.NewToolchainObject(rawObject.Object)
		if err != nil {
			return nil, err
		}
		objects[i] = toolchainObject
	}
	return objects, nil
}
