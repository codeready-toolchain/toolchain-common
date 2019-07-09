package condition

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddOrUpdateStatusConditions appends the new conditions to the condition slice. If there is already a condition
// with the same type in the current condition array then the condition is updated in the result slice.
// If the condition is not changed then the same unmodified slice is returned.
// Also returns a bool flag which indicates if the conditions where updated/added
// TODO: move to toolchain-common
func AddOrUpdateStatusConditions(conditions []toolchainv1alpha1.Condition, newConditions ...toolchainv1alpha1.Condition) ([]toolchainv1alpha1.Condition, bool) {
	var atLeastOneUpdated bool
	var updated bool
	for _, cond := range newConditions {
		conditions, updated = addOrUpdateStatusCondition(conditions, cond)
		atLeastOneUpdated = atLeastOneUpdated || updated
	}

	return conditions, atLeastOneUpdated
}

// TODO: move to toolchain-common
func addOrUpdateStatusCondition(conditions []toolchainv1alpha1.Condition, newCondition toolchainv1alpha1.Condition) ([]toolchainv1alpha1.Condition, bool) {
	newCondition.LastTransitionTime = metav1.Now()

	if conditions == nil {
		return []toolchainv1alpha1.Condition{newCondition}, true
	} else {
		for i, cond := range conditions {
			if cond.Type == newCondition.Type {
				// Condition already present. Update it if needed.
				if cond.Status == newCondition.Status &&
					cond.Reason == newCondition.Reason &&
					cond.Message == newCondition.Message {
					// Nothing changed. No need to update.
					return conditions, false
				}

				// Update LastTransitionTime only if the status changed otherwise keep the old time
				if newCondition.Status == cond.Status {
					newCondition.LastTransitionTime = cond.LastTransitionTime
				}
				// Don't modify the currentConditions slice. Generate a new slice instead.
				res := make([]toolchainv1alpha1.Condition, len(conditions))
				copy(res, conditions)
				res[i] = newCondition
				return res, true
			}
		}
	}
	return append(conditions, newCondition), true
}
