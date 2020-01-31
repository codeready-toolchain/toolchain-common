package controller

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// EnqueueRequestForOwnerByLabel an event handler will convert events on a resource to requests on
// another resource whose name if found in a given label
type EnqueueRequestForOwnerByLabel struct {
	handler.EnqueueRequestsFromMapFunc
	Namespace string
	Label     string
}

var _ handler.EventHandler = &EnqueueRequestForOwnerByLabel{}

// Map maps the namespace to a request on the "owner" (or "associated") NSTemplateSet
func (m EnqueueRequestForOwnerByLabel) Map(obj handler.MapObject) []reconcile.Request {
	if name, exists := obj.Meta.GetLabels()[m.Label]; exists {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{Namespace: m.Namespace, Name: name},
			},
		}
	}
	// the obj was not a namespace or it did not have the required label.
	return []reconcile.Request{}
}
