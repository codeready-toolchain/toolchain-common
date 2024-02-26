package spaceprovisionerconfig

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestReadyPredicate(t *testing.T) {
	t.Run("matching", func(t *testing.T) {
		// given
		pred := &ready{}
		spc := NewSpaceProvisionerConfig("spc", "default", WithReadyConditionValid())

		// when & then
		assert.True(t, pred.Matches(spc))
	})

	t.Run("fixer with no conditions", func(t *testing.T) {
		// given
		pred := &ready{}
		spc := NewSpaceProvisionerConfig("spc", "default")

		// when
		spc = pred.FixToMatch(spc)

		// then
		assert.True(t, condition.IsTrue(spc.Status.Conditions, toolchainv1alpha1.ConditionReady))
	})
	t.Run("fixer with different conditions", func(t *testing.T) {
		// given
		pred := &ready{}
		spc := NewSpaceProvisionerConfig("spc", "default")
		spc.Status.Conditions = []toolchainv1alpha1.Condition{
			{
				Type:   toolchainv1alpha1.ConditionType("made up"),
				Status: corev1.ConditionTrue,
			},
		}

		// when
		spc = pred.FixToMatch(spc)

		// then
		assert.True(t, condition.IsTrue(spc.Status.Conditions, toolchainv1alpha1.ConditionReady))
		assert.Len(t, spc.Status.Conditions, 2)
	})
	t.Run("fixer with wrong condition", func(t *testing.T) {
		// given
		pred := &ready{}
		spc := NewSpaceProvisionerConfig("spc", "default", WithReadyConditionInvalid("because"))

		// when
		spc = pred.FixToMatch(spc)

		// then
		assert.True(t, condition.IsTrueWithReason(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, toolchainv1alpha1.SpaceProvisionerConfigValidReason))
	})
}

func TestNotReadyPredicate(t *testing.T) {
	t.Run("matching", func(t *testing.T) {
		// given
		pred := &notReady{}
		spc := NewSpaceProvisionerConfig("spc", "default", WithReadyConditionInvalid("any reason"))

		// when & then
		assert.True(t, pred.Matches(spc))
	})

	t.Run("fixer with no conditions", func(t *testing.T) {
		// given
		pred := &notReady{}
		spc := NewSpaceProvisionerConfig("spc", "default")

		// when
		spc = pred.FixToMatch(spc)

		// then
		assert.True(t, condition.IsFalse(spc.Status.Conditions, toolchainv1alpha1.ConditionReady))
	})
	t.Run("fixer with different conditions", func(t *testing.T) {
		// given
		pred := &notReady{}
		spc := NewSpaceProvisionerConfig("spc", "default")
		spc.Status.Conditions = []toolchainv1alpha1.Condition{
			{
				Type:   toolchainv1alpha1.ConditionType("made up"),
				Status: corev1.ConditionTrue,
			},
		}

		// when
		spc = pred.FixToMatch(spc)

		// then
		assert.True(t, condition.IsFalse(spc.Status.Conditions, toolchainv1alpha1.ConditionReady))
		assert.Len(t, spc.Status.Conditions, 2)
	})
	t.Run("fixer with wrong condition", func(t *testing.T) {
		// given
		pred := &notReady{}
		spc := NewSpaceProvisionerConfig("spc", "default", WithReadyConditionValid())

		// when
		spc = pred.FixToMatch(spc)

		// then
		assert.True(t, condition.IsFalse(spc.Status.Conditions, toolchainv1alpha1.ConditionReady))
	})
}

func TestNotReadyWithReasonPredicate(t *testing.T) {
	t.Run("matching", func(t *testing.T) {
		// given
		pred := &notReadyWithReason{expectedReason: "the right reason"}
		spc := NewSpaceProvisionerConfig("spc", "default", WithReadyConditionInvalid("the right reason"))

		// when & then
		assert.True(t, pred.Matches(spc))
	})

	t.Run("fixer with no conditions", func(t *testing.T) {
		// given
		pred := &notReadyWithReason{expectedReason: "the right reason"}
		spc := NewSpaceProvisionerConfig("spc", "default")

		// when
		spc = pred.FixToMatch(spc)

		// then
		assert.True(t, condition.IsFalseWithReason(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, "the right reason"))
	})
	t.Run("fixer with different conditions", func(t *testing.T) {
		// given
		pred := &notReadyWithReason{expectedReason: "the right reason"}
		spc := NewSpaceProvisionerConfig("spc", "default")
		spc.Status.Conditions = []toolchainv1alpha1.Condition{
			{
				Type:   toolchainv1alpha1.ConditionType("made up"),
				Status: corev1.ConditionTrue,
			},
		}

		// when
		spc = pred.FixToMatch(spc)

		// then
		assert.True(t, condition.IsFalseWithReason(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, "the right reason"))
		assert.Len(t, spc.Status.Conditions, 2)
	})
	t.Run("fixer with wrong condition", func(t *testing.T) {
		// given
		pred := &notReadyWithReason{expectedReason: "the right reason"}
		spc := NewSpaceProvisionerConfig("spc", "default", WithReadyConditionInvalid("the wrong reason"))

		// when
		spc = pred.FixToMatch(spc)

		// then
		assert.True(t, condition.IsFalseWithReason(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, "the right reason"))
	})
}
