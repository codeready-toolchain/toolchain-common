package controller

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// LabelMapper a mapper that will convert events on a resource to requests on
// another resource whose name if found in a given label
type LabelMapper struct {
	namespace string
	label     string
}

// NewLabelMapper returns a new mapper to convert a reconcile request on a given resource (source)
// to a reconcile request on another resource whose name is found in labels of the source resource
// and which is located in the given namespace (or "" for cluster-wide resources)
func NewLabelMapper(namespace string, label string) LabelMapper {
	return LabelMapper{
		namespace: namespace,
		label:     label,
	}
}

var _ handler.Mapper = LabelMapper{}

// Map maps the namespace to a request on the "owner" (or "associated") NSTemplateSet
func (m LabelMapper) Map(obj handler.MapObject) []reconcile.Request {
	if name, exists := obj.Meta.GetLabels()[m.label]; exists {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{Namespace: m.namespace, Name: name},
			},
		}
	}
	// the obj was not a namespace or it did not have the required label.
	return []reconcile.Request{}
}
