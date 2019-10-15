package condition_test

import (
	"fmt"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddOrUpdateStatusConditions(t *testing.T) {
	t.Run("no new conditions", func(t *testing.T) {
		// given
		current := existingConditions(3)
		//when
		newCs, updated := condition.AddOrUpdateStatusConditions(current)
		// then
		assert.False(t, updated)
		assert.Equal(t, current, newCs)
	})

	t.Run("add to empty condition slice", func(t *testing.T) {
		//when
		newConds := newConditions(1)
		result, updated := condition.AddOrUpdateStatusConditions(existingConditions(0), newConds...)
		// then
		assert.True(t, updated)
		test.AssertConditionsMatch(t, result, newConds...)
	})

	t.Run("add new conditions", func(t *testing.T) {
		// given
		current := existingConditions(5)
		//when
		newConds := newConditions(3)
		result, updated := condition.AddOrUpdateStatusConditions(current, newConds...)
		// then
		assert.True(t, updated)
		test.AssertConditionsMatch(t, result, append(current, newConds...)...)
	})

	t.Run("update conditions", func(t *testing.T) {
		// given
		current := existingConditions(5)
		//when
		newConds := []toolchainv1alpha1.Condition{
			// Updated message
			{
				Type:    current[1].Type,
				Message: "Updated message",
				Status:  current[1].Status,
				Reason:  current[1].Reason,
			},
			// Updated status
			{
				Type:    current[2].Type,
				Message: current[2].Message,
				Status:  reverseStatus(current[2].Status),
				Reason:  current[2].Reason,
			},
			// Updated reason
			{
				Type:    current[3].Type,
				Message: current[3].Message,
				Status:  current[3].Status,
				Reason:  "UpdatedReason",
			},
			// Nothing changed
			{
				Type:    current[4].Type,
				Message: current[4].Message,
				Status:  current[4].Status,
				Reason:  current[4].Reason,
			},
		}
		result, updated := condition.AddOrUpdateStatusConditions(current, newConds...)
		// then
		assert.True(t, updated)
		// 1st and the 5th are from the current condition slice and 2-3 are from the new one
		test.AssertConditionsMatch(t, result, []toolchainv1alpha1.Condition{current[0], newConds[0], newConds[1], newConds[2], current[4]}...)
		// Check the LastTransitionTime. Should be changed in 3rd only where we updated the status.
		for i, c := range current {
			if i != 2 {
				assert.NotEmpty(t, c.LastTransitionTime)
				assert.Equal(t, c.LastTransitionTime, result[i].LastTransitionTime)
			} else {
				assert.True(t, c.LastTransitionTime.Before(&result[i].LastTransitionTime))
			}
		}
	})
}

func TestFindConditionByType(t *testing.T) {
	conditions := []toolchainv1alpha1.Condition{
		{
			Type:   toolchainv1alpha1.ConditionReady,
			Status: corev1.ConditionTrue,
		},
	}

	t.Run("found", func(t *testing.T) {
		got, found := condition.FindConditionByType(conditions, toolchainv1alpha1.ConditionReady)
		assert.True(t, found)
		assert.Equal(t, toolchainv1alpha1.ConditionReady, got.Type)
	})

	t.Run("not_found", func(t *testing.T) {
		got, found := condition.FindConditionByType(conditions, toolchainv1alpha1.ConditionType("Completed"))
		assert.False(t, found)
		assert.Equal(t, toolchainv1alpha1.Condition{}, got)
	})
}

func TestIsTrue(t *testing.T) {
	conditions := []toolchainv1alpha1.Condition{
		{
			Type:   toolchainv1alpha1.ConditionReady,
			Status: corev1.ConditionTrue,
		},
		{
			Type:   "custom_bool",
			Status: corev1.ConditionTrue,
		},
		{
			Type:   "false_cond",
			Status: corev1.ConditionFalse,
		},
		{
			Type:   "unknown_cond",
			Status: corev1.ConditionUnknown,
		},
		{
			Type:   "something_else",
			Status: "someStatus",
		},
	}

	t.Run("ready set to true", func(t *testing.T) {
		assert.True(t, condition.IsTrue(conditions, toolchainv1alpha1.ConditionReady))
	})

	t.Run("custom bool set to true", func(t *testing.T) {
		assert.True(t, condition.IsTrue(conditions, "custom_bool"))
	})

	t.Run("false", func(t *testing.T) {
		assert.False(t, condition.IsTrue(conditions, "false_cond"))
	})

	t.Run("explicitly unknown", func(t *testing.T) {
		assert.False(t, condition.IsTrue(conditions, "unknown_cond"))
	})

	t.Run("unknown", func(t *testing.T) {
		assert.False(t, condition.IsTrue(conditions, "unknown"))
	})

	t.Run("status is not bool", func(t *testing.T) {
		assert.False(t, condition.IsTrue(conditions, "something_else"))
	})
}

func existingConditions(size int) []toolchainv1alpha1.Condition {
	return conditions(size, "Existing")
}

func newConditions(size int) []toolchainv1alpha1.Condition {
	return conditions(size, "New")
}

func conditions(size int, prefix string) []toolchainv1alpha1.Condition {
	conditions := make([]toolchainv1alpha1.Condition, size)
	for i := 0; i < size; i++ {
		conditions[i] = toolchainv1alpha1.Condition{
			Type:    toolchainv1alpha1.ConditionType(fmt.Sprintf("%sTestConditionType%d", prefix, i)),
			Message: fmt.Sprintf("%s Message %d", prefix, i),
			Status:  apiv1.ConditionTrue,
			Reason:  fmt.Sprintf("%sReason%d", prefix, i),
		}
		if prefix == "Existing" {
			conditions[i].LastTransitionTime = metav1.NewTime(time.Now().Add(-time.Second)) // one second ago
		}
	}
	return conditions
}

func reverseStatus(status apiv1.ConditionStatus) apiv1.ConditionStatus {
	switch status {
	case apiv1.ConditionTrue:
		return apiv1.ConditionFalse
	case apiv1.ConditionFalse:
		return apiv1.ConditionTrue
	}
	// Unknown
	return apiv1.ConditionFalse
}
