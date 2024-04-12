package template

import (
	"fmt"
	"math/rand"
	"time"

	templatev1 "github.com/openshift/api/template/v1"
	"github.com/openshift/library-go/pkg/template/generator"
	"github.com/openshift/library-go/pkg/template/templateprocessing"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
func (p Processor) Process(tmpl *templatev1.Template, values map[string]string, filters ...FilterFunc) ([]runtimeclient.Object, error) {
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
		"expression": generator.NewExpressionValueGenerator(rand.New(rand.NewSource(time.Now().UnixNano()))), //nolint:gosec
	})
	if err := tmplProcessor.Process(tmpl); len(err) > 0 {
		return nil, errors.Wrap(err.ToAggregate(), "unable to process template")
	}
	var result templatev1.Template
	if err := p.scheme.Convert(tmpl, &result, nil); err != nil {
		return nil, errors.Wrap(err, "failed to convert template to external template object")
	}
	filtered := Filter(result.Objects, filters...)
	objects := make([]runtimeclient.Object, len(filtered))
	for i, rawObject := range filtered {
		clientObj, ok := rawObject.Object.(runtimeclient.Object)
		if !ok {
			return nil, fmt.Errorf("unable to cast of the object to client.Object: %+v", rawObject)
		}
		objects[i] = clientObj
	}
	return objects, nil
}
