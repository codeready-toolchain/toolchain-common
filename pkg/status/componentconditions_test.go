package status

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
)

func TestValidateComponentConditionReady(t *testing.T) {

	conditionReady := toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionTrue,
		Reason:  "DeploymentReady",
		Message: "",
	}

	conditionNotReady := toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  "DeploymentNotReady",
		Message: "deployment has unready status conditions: Available",
	}

	conditionOtherType := toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.MasterUserRecordProvisioning,
		Status:  corev1.ConditionTrue,
		Reason:  "Provisioned",
		Message: "",
	}

	conditionOtherType2 := toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.MasterUserRecordUserProvisionedNotificationCreated,
		Status:  corev1.ConditionTrue,
		Reason:  "NotificationCreated",
		Message: "",
	}

	t.Run("single conditions", func(t *testing.T) {

		t.Run("no ready condition", func(t *testing.T) {
			err := ValidateComponentConditionReady(conditionOtherType)
			require.Error(t, err)
			assert.EqualError(t, err, "a ready condition was not found")
		})

		t.Run("condition not ready", func(t *testing.T) {
			err := ValidateComponentConditionReady(conditionNotReady)
			require.Error(t, err)
			assert.EqualError(t, err, "deployment has unready status conditions: Available")
		})

		t.Run("condition ready", func(t *testing.T) {
			err := ValidateComponentConditionReady(conditionReady)
			assert.NoError(t, err)
		})
	})

	t.Run("multiple conditions", func(t *testing.T) {

		t.Run("multiple - no ready condition", func(t *testing.T) {
			conditions := []toolchainv1alpha1.Condition{conditionOtherType, conditionOtherType2}
			err := ValidateComponentConditionReady(conditions...)
			require.Error(t, err)
			assert.EqualError(t, err, "a ready condition was not found")
		})

		t.Run("multiple - condition not ready", func(t *testing.T) {
			conditions := []toolchainv1alpha1.Condition{conditionNotReady, conditionOtherType}
			err := ValidateComponentConditionReady(conditions...)
			require.Error(t, err)
			assert.EqualError(t, err, "deployment has unready status conditions: Available")
		})

		t.Run("multiple - condition ready", func(t *testing.T) {
			conditions := []toolchainv1alpha1.Condition{conditionReady, conditionOtherType}
			err := ValidateComponentConditionReady(conditions...)
			require.NoError(t, err)
		})
	})
}
