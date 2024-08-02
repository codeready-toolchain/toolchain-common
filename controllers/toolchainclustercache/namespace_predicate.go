package toolchainclustercache

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// namespacePredicate will filter out all events out of the provided namespace
type namespacePredicate struct {
	namespace string
}

// Update allows events only in the given namespace
func (p namespacePredicate) Update(e event.UpdateEvent) bool {
	return e.ObjectNew.GetNamespace() == p.namespace
}

// Create allows events only in the given namespace
func (p namespacePredicate) Create(e event.CreateEvent) bool {
	return e.Object.GetNamespace() == p.namespace
}

// Delete allows events only in the given namespace
func (p namespacePredicate) Delete(e event.DeleteEvent) bool {
	return e.Object.GetNamespace() == p.namespace
}

// Generic allows events only in the given namespace
func (p namespacePredicate) Generic(e event.GenericEvent) bool {
	return e.Object.GetNamespace() == p.namespace
}
