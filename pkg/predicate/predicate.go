package predicate

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var log = logf.Log.WithName("generation_not_changed_predicate").WithName("eventFilters")

// EitherUpdateWhenGenerationNotChangedOrDelete implements a predicate that triggers
// reconciles either for updates when generation was not changed or for deletion
type EitherUpdateWhenGenerationNotChangedOrDelete struct {
}

// Update implements default UpdateEvent filter for validating no generation change
func (EitherUpdateWhenGenerationNotChangedOrDelete) Update(e event.UpdateEvent) bool {
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
func (EitherUpdateWhenGenerationNotChangedOrDelete) Create(_ event.CreateEvent) bool {
	return false
}

// Delete implements Predicate
func (EitherUpdateWhenGenerationNotChangedOrDelete) Delete(_ event.DeleteEvent) bool {
	return true
}

// Generic implements Predicate
func (EitherUpdateWhenGenerationNotChangedOrDelete) Generic(_ event.GenericEvent) bool {
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
