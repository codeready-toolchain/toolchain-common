package predicate

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var log = logf.Log.WithName("generation_not_changed_predicate").WithName("eventFilters")

// OnlyUpdateWhenGenerationNotChanged implements a default update predicate function on no generation change
// (adapted from sigs.k8s.io/controller-runtime/pkg/predicate/predicate.ResourceVersionChangedPredicate)
// other predicate functions return false for all cases
// Copied and slightly modified from github.com/operator-framework/operator-sdk/pkg/predicate/predicate.go
type OnlyUpdateWhenGenerationNotChanged struct {
}

// Update implements default UpdateEvent filter for validating no generation change
func (OnlyUpdateWhenGenerationNotChanged) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		log.Error(nil, "Update event has no old runtime object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		log.Error(nil, "Update event has no new runtime object for update", "event", e)
		return false
	}
	return e.ObjectNew.GetGeneration() == e.ObjectOld.GetGeneration()
}

// Create implements Predicate
func (OnlyUpdateWhenGenerationNotChanged) Create(e event.CreateEvent) bool {
	return false
}

// Delete implements Predicate
func (OnlyUpdateWhenGenerationNotChanged) Delete(e event.DeleteEvent) bool {
	return false
}

// Generic implements Predicate
func (OnlyUpdateWhenGenerationNotChanged) Generic(e event.GenericEvent) bool {
	return false
}

// LabelsAndGenerationPredicate is based on the default predicate functions but overrides the Update function
// to only return true if either the labels or generation have changed, status changes won't cause reconciliation
type LabelsAndGenerationPredicate struct {
	predicate.Funcs
}

// Update only returns true if either the labels or generation have changed
func (LabelsAndGenerationPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		log.Error(nil, "Update event has no old runtime object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		log.Error(nil, "Update event has no new runtime object for update", "event", e)
		return false
	}
	// reconcile if the labels have changed
	return !reflect.DeepEqual(e.ObjectOld.GetLabels(), e.ObjectNew.GetLabels()) ||
		e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
}
