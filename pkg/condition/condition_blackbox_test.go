package condition_test

import (
	"fmt"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		t.Run("with LastUpdatedTime", func(t *testing.T) {
			//when
			newCs := condition.AddOrUpdateStatusConditionsWithLastUpdatedTimestamp(current)
			// then
			assert.Equal(t, current, newCs)
		})
	})

	t.Run("add to empty condition slice", func(t *testing.T) {
		//when
		newConds := newConditions(1)
		result, updated := condition.AddOrUpdateStatusConditions(existingConditions(0), newConds...)
		// then
		assert.True(t, updated)
		test.AssertConditionsMatch(t, result, newConds...)

		t.Run("with LastUpdatedTime", func(t *testing.T) {
			//when
			result := condition.AddOrUpdateStatusConditionsWithLastUpdatedTimestamp(existingConditions(0), newConds...)
			// then
			test.AssertConditionsMatch(t, result, newConds...)
		})
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

		t.Run("with LastUpdatedTime", func(t *testing.T) {
			//when
			newConds := newConditions(3)
			result := condition.AddOrUpdateStatusConditionsWithLastUpdatedTimestamp(current, newConds...)
			// then
			test.AssertConditionsMatch(t, result, append(current, newConds...)...)
		})
	})

	t.Run("update conditions", func(t *testing.T) {
		// given
		current := existingConditions(5)
		for i := range current {
			current[i].LastUpdatedTime = &current[i].LastTransitionTime
		}
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

		t.Run("with LastUpdatedTime", func(t *testing.T) {
			//when
			result := condition.AddOrUpdateStatusConditionsWithLastUpdatedTimestamp(current, newConds...)
			// then
			// all new conditions are modified
			test.AssertConditionsMatch(t, result, []toolchainv1alpha1.Condition{current[0], newConds[0], newConds[1], newConds[2], newConds[3]}...)
			// Check the LastTransitionTime. Should be changed in 3rd only where we updated the status.
			// All current conditions should have LastUpdatedTime changed
			for i, c := range current {
				if i != 2 {
					assert.NotEmpty(t, c.LastTransitionTime)
					assert.Equal(t, c.LastTransitionTime, result[i].LastTransitionTime)
				} else {
					assert.True(t, c.LastTransitionTime.Before(&result[i].LastTransitionTime))
				}
			}
			// Check all new conditions should have LastUpdatedTime changed
			for i := 1; i < len(result); i++ {
				require.NotEmpty(t, result[i].LastUpdatedTime)
				assert.True(t, result[i].LastUpdatedTime.After(current[i].LastUpdatedTime.Time))
			}
			assert.False(t, result[0].LastUpdatedTime.After(current[0].LastUpdatedTime.Time)) // The existing not updated condition is not affected
		})
	})
}

func TestAddStatusConditions(t *testing.T) {

	t.Run("without duplicate types", func(t *testing.T) {
		// given
		conditions := []toolchainv1alpha1.Condition{
			{
				Type:               "foo",
				Message:            "message",
				Reason:             "reason",
				LastTransitionTime: metav1.Now(),
			},
		}
		c := toolchainv1alpha1.Condition{
			Type:    "bar",
			Message: "message",
			Reason:  "reason",
		}
		// when
		result := condition.AddStatusConditions(conditions, c)
		// then
		require.Len(t, result, 2)
		assert.Equal(t, toolchainv1alpha1.ConditionType("foo"), result[0].Type)
		assert.Equal(t, toolchainv1alpha1.ConditionType("bar"), result[1].Type)
		assert.False(t, result[1].LastTransitionTime.IsZero())
	})

	t.Run("without setting LastTransitionTime", func(t *testing.T) {
		// given
		conditions := []toolchainv1alpha1.Condition{
			{
				Type:               "foo",
				Message:            "message",
				Reason:             "reason",
				LastTransitionTime: metav1.Now(),
			},
		}
		oneMinuteAgo := time.Now().Add(-1 * time.Minute)
		c := toolchainv1alpha1.Condition{
			Type:               "bar",
			Message:            "message",
			Reason:             "reason",
			LastTransitionTime: metav1.NewTime(oneMinuteAgo),
		}
		// when
		result := condition.AddStatusConditions(conditions, c)
		// then
		require.Len(t, result, 2)
		assert.Equal(t, toolchainv1alpha1.ConditionType("foo"), result[0].Type)
		assert.Equal(t, toolchainv1alpha1.ConditionType("bar"), result[1].Type)
		assert.Equal(t, metav1.NewTime(oneMinuteAgo), result[1].LastTransitionTime)
	})

	t.Run("with duplicate types", func(t *testing.T) {
		// given
		conditions := []toolchainv1alpha1.Condition{
			{
				Type:    "foo",
				Message: "message",
				Reason:  "reason",
			},
		}
		c := toolchainv1alpha1.Condition{
			Type:    "foo",
			Message: "message",
			Reason:  "reason",
		}
		// when
		result := condition.AddStatusConditions(conditions, c)
		// then
		require.Len(t, result, 2)
		assert.Equal(t, toolchainv1alpha1.ConditionType("foo"), result[0].Type)
		assert.Equal(t, toolchainv1alpha1.ConditionType("foo"), result[1].Type)
		assert.False(t, result[1].LastTransitionTime.IsZero())
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

func TestIsFalse(t *testing.T) {
	conditions := []toolchainv1alpha1.Condition{
		{
			Type:   toolchainv1alpha1.ConditionReady,
			Status: corev1.ConditionFalse,
		},
		{
			Type:   "custom_bool",
			Status: corev1.ConditionFalse,
		},
		{
			Type:   "false_cond",
			Status: corev1.ConditionTrue,
		},
		{
			Type:   "unknown_cond",
			Status: corev1.ConditionUnknown,
		},
		{
			Type:   "something_else",
			Status: "someStatus",
		},
		{
			Type:   toolchainv1alpha1.ToolchainStatusUnreadyNotificationCreated,
			Status: corev1.ConditionFalse,
			Reason: toolchainv1alpha1.ToolchainStatusRestoredNotificationCRCreationFailedReason,
		},
	}
	t.Run("ready set to false", func(t *testing.T) {
		assert.True(t, condition.IsFalse(conditions, toolchainv1alpha1.ConditionReady))
	})

	t.Run("toolchainstatus unready notification set to false", func(t *testing.T) {
		assert.True(t, condition.IsFalse(conditions, toolchainv1alpha1.ToolchainStatusUnreadyNotificationCreated))
	})

	t.Run("custom bool set to false", func(t *testing.T) {
		assert.True(t, condition.IsFalse(conditions, "custom_bool"))
	})

	t.Run("assert false when status is true", func(t *testing.T) {
		assert.False(t, condition.IsFalse(conditions, "false_cond"))
	})

	t.Run("explicitly unknown status", func(t *testing.T) {
		assert.False(t, condition.IsFalse(conditions, "unknown_cond"))
	})

	t.Run("unknown", func(t *testing.T) {
		assert.False(t, condition.IsFalse(conditions, "unknown"))
	})

	t.Run("status is not bool", func(t *testing.T) {
		assert.False(t, condition.IsFalse(conditions, "something_else"))
	})
}

func TestIsFalseWithReason(t *testing.T) {
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
			Type:   "false_cond_without_reason",
			Status: corev1.ConditionFalse,
		},
		{
			Type:   "false_cond_with_reason",
			Status: corev1.ConditionFalse,
			Reason: "reason",
		},
		{
			Type:   "false_cond_with_other_reason",
			Status: corev1.ConditionFalse,
			Reason: "other",
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
		assert.False(t, condition.IsFalseWithReason(conditions, toolchainv1alpha1.ConditionReady, "reason"))
	})

	t.Run("custom bool set to true", func(t *testing.T) {
		assert.False(t, condition.IsFalseWithReason(conditions, "custom_bool", "reason"))
	})

	t.Run("false without reason", func(t *testing.T) {
		assert.False(t, condition.IsFalseWithReason(conditions, "false_cond_without_reason", "reason"))
	})

	t.Run("false with reason", func(t *testing.T) {
		assert.True(t, condition.IsFalseWithReason(conditions, "false_cond_with_reason", "reason"))
	})

	t.Run("false with other reason", func(t *testing.T) {
		assert.False(t, condition.IsFalseWithReason(conditions, "false_cond_with_other_reason", "reason"))
	})

	t.Run("explicitly unknown", func(t *testing.T) {
		assert.False(t, condition.IsFalseWithReason(conditions, "unknown_cond", "reason"))
	})

	t.Run("unknown", func(t *testing.T) {
		assert.False(t, condition.IsFalseWithReason(conditions, "unknown", "reason"))
	})

	t.Run("status is not bool", func(t *testing.T) {
		assert.False(t, condition.IsFalseWithReason(conditions, "something_else", "reason"))
	})
}

func TestIsTrueWithReason(t *testing.T) {
	conditions := []toolchainv1alpha1.Condition{
		{
			Type:   toolchainv1alpha1.ConditionReady,
			Status: corev1.ConditionFalse,
		},
		{
			Type:   "custom_bool",
			Status: corev1.ConditionFalse,
		},
		{
			Type:   "true_cond_without_reason",
			Status: corev1.ConditionTrue,
		},
		{
			Type:   "true_cond_with_reason",
			Status: corev1.ConditionTrue,
			Reason: "reason",
		},
		{
			Type:   "true_cond_with_other_reason",
			Status: corev1.ConditionTrue,
			Reason: "other",
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
		assert.False(t, condition.IsTrueWithReason(conditions, toolchainv1alpha1.ConditionReady, "reason"))
	})

	t.Run("custom bool set to true", func(t *testing.T) {
		assert.False(t, condition.IsTrueWithReason(conditions, "custom_bool", "reason"))
	})

	t.Run("false without reason", func(t *testing.T) {
		assert.False(t, condition.IsTrueWithReason(conditions, "true_cond_without_reason", "reason"))
	})

	t.Run("false with reason", func(t *testing.T) {
		assert.True(t, condition.IsTrueWithReason(conditions, "true_cond_with_reason", "reason"))
	})

	t.Run("false with other reason", func(t *testing.T) {
		assert.False(t, condition.IsTrueWithReason(conditions, "true_cond_with_other_reason", "reason"))
	})

	t.Run("explicitly unknown", func(t *testing.T) {
		assert.False(t, condition.IsTrueWithReason(conditions, "unknown_cond", "reason"))
	})

	t.Run("unknown", func(t *testing.T) {
		assert.False(t, condition.IsTrueWithReason(conditions, "unknown", "reason"))
	})

	t.Run("status is not bool", func(t *testing.T) {
		assert.False(t, condition.IsTrueWithReason(conditions, "something_else", "reason"))
	})
}

func TestHasConditionReason(t *testing.T) {
	conditions := []toolchainv1alpha1.Condition{
		{
			Type:   toolchainv1alpha1.ConditionReady,
			Status: corev1.ConditionFalse,
			Reason: "Disabled",
		},
		{
			Type:   toolchainv1alpha1.ConditionReady,
			Status: corev1.ConditionFalse,
			Reason: "Unknown",
		},
	}

	t.Run("found", func(t *testing.T) {
		assert.True(t, condition.HasConditionReason(conditions, toolchainv1alpha1.ConditionReady, "Disabled"))
	})

	t.Run("not found", func(t *testing.T) {
		assert.False(t, condition.HasConditionReason(conditions, toolchainv1alpha1.ConditionReady, "Testing"))
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
	case apiv1.ConditionTrue, apiv1.ConditionUnknown:
		return apiv1.ConditionFalse
	case apiv1.ConditionFalse:
		return apiv1.ConditionTrue
	default:
		// Unknown
		return apiv1.ConditionFalse
	}
}
