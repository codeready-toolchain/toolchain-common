package controller

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NewEnqueueRequestForOwnerByLabel returns an event handler will convert events on a resource to requests on
// another resource whose name if found in a given label
func NewEnqueueRequestForOwnerByLabel(namespace, label string) handler.EventHandler {
	return &handler.EnqueueRequestsFromMapFunc{
		ToRequests: &eventToOwnerByLabelMapper{
			label:     label,
			namespace: namespace,
		},
	}
}

var _ handler.Mapper = &eventToOwnerByLabelMapper{}

// eventToOwnerByLabelMapper implementation of an handler mapper which
// returns a reconcile request for the resource in the given namespace
// and whose name is in the given label
type eventToOwnerByLabelMapper struct {
	namespace string
	label     string
}

// Map maps the namespace to a request on the "owner" (or "associated") resource
// (if the label exists)
func (m eventToOwnerByLabelMapper) Map(obj handler.MapObject) []reconcile.Request {
	if name, exists := obj.Meta.GetLabels()[m.label]; exists {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: m.namespace,
					Name:      name,
				},
			},
		}
	}
	// the obj was not a namespace or it did not have the required label.
	return []reconcile.Request{}
}
