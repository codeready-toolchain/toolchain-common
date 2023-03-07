package test

import (
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConditionsMatch(t *testing.T) {

	t.Run("conditions match", func(t *testing.T) {
		t.Run("empty lists", func(t *testing.T) {
			assert.True(t, ConditionsMatch(nil))
			assert.True(t, ConditionsMatch([]toolchainv1alpha1.Condition{}))
		})

		t.Run("conditions match", func(t *testing.T) {
			assert.True(t, ConditionsMatch(actual(),
				toolchainv1alpha1.Condition{
					Type:               testType2,
					Status:             apiv1.ConditionFalse,
					Reason:             "reason2",
					LastTransitionTime: metav1.NewTime(time.Now()),
				},
				toolchainv1alpha1.Condition{
					Type:   testType3,
					Status: apiv1.ConditionUnknown,
				},
				toolchainv1alpha1.Condition{
					Type:    testType1,
					Status:  apiv1.ConditionTrue,
					Reason:  "reason1",
					Message: "message1",
				},
			))
		})

	})

	t.Run("conditions don't match", func(t *testing.T) {
		t.Run("different sizes of the lists", func(t *testing.T) {
			assert.False(t, ConditionsMatch(nil, toolchainv1alpha1.Condition{}))
			conditions := actual()
			assert.False(t, ConditionsMatch(conditions))
			assert.False(t, ConditionsMatch(conditions, conditions[0]))
			assert.False(t, ConditionsMatch(conditions, conditions[0], conditions[1], conditions[2], conditions[0]))
		})
		t.Run("missing conditions", func(t *testing.T) {
			conditions := actual()
			assert.False(t, ConditionsMatch(conditions, conditions[0], conditions[0], conditions[0]))
		})
		t.Run("wrong conditions", func(t *testing.T) {
			conditions := actual()
			assert.False(t, ConditionsMatch(conditions,
				conditions[0],
				conditions[1],
				toolchainv1alpha1.Condition{
					Type:    testType3,
					Status:  apiv1.ConditionUnknown,
					Message: "unexpected message",
				},
			))
		})
	})
}

func TestContainsCondition(t *testing.T) {

	t.Run("LastTransitionTime ignored", func(t *testing.T) {
		c, found := ContainsCondition(actual(),
			toolchainv1alpha1.Condition{
				Type:    testType1,
				Status:  apiv1.ConditionTrue,
				Reason:  "reason1",
				Message: "message1",
			})
		assert.True(t, found)
		assert.NotEmpty(t, c.LastTransitionTime)
	})

	t.Run("doesn't contain condition", func(t *testing.T) {
		conditions := actual()
		t.Run("empty list", func(t *testing.T) {
			_, found := ContainsCondition(nil, toolchainv1alpha1.Condition{})
			assert.False(t, found)
			_, found = ContainsCondition([]toolchainv1alpha1.Condition{}, toolchainv1alpha1.Condition{})
			assert.False(t, found)
		})
		t.Run("missing type", func(t *testing.T) {
			var testType toolchainv1alpha1.ConditionType = "unknown"
			_, found := ContainsCondition(conditions,
				toolchainv1alpha1.Condition{
					Type:    testType,
					Status:  conditions[0].Status,
					Reason:  conditions[0].Reason,
					Message: conditions[0].Message,
				})
			assert.False(t, found)
		})
		t.Run("status doesn't match", func(t *testing.T) {
			_, found := ContainsCondition(conditions,
				toolchainv1alpha1.Condition{
					Type:    conditions[0].Type,
					Status:  apiv1.ConditionUnknown,
					Reason:  conditions[0].Reason,
					Message: conditions[0].Message,
				})
			assert.False(t, found)
		})
		t.Run("reason doesn't match", func(t *testing.T) {
			_, found := ContainsCondition(conditions,
				toolchainv1alpha1.Condition{
					Type:    conditions[0].Type,
					Status:  conditions[0].Status,
					Message: conditions[0].Message,
				})
			assert.False(t, found)
		})
		t.Run("message doesn't match", func(t *testing.T) {
			_, found := ContainsCondition(conditions,
				toolchainv1alpha1.Condition{
					Type:   conditions[0].Type,
					Status: conditions[0].Status,
					Reason: conditions[0].Reason,
				})
			assert.False(t, found)
		})
	})
}

const (
	testType1 toolchainv1alpha1.ConditionType = "test1"
	testType2 toolchainv1alpha1.ConditionType = "test2"
	testType3 toolchainv1alpha1.ConditionType = "test3"
)

func actual() []toolchainv1alpha1.Condition {
	return []toolchainv1alpha1.Condition{
		{
			Type:               testType1,
			Status:             apiv1.ConditionTrue,
			Reason:             "reason1",
			Message:            "message1",
			LastTransitionTime: metav1.NewTime(time.Now().Add(-time.Second)),
		},
		{
			Type:               testType2,
			Status:             apiv1.ConditionFalse,
			Reason:             "reason2",
			LastTransitionTime: metav1.NewTime(time.Now().Add(-time.Minute)),
		},
		{
			Type:   testType3,
			Status: apiv1.ConditionUnknown,
		},
	}
}
