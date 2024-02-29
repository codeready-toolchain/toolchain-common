package spaceprovisionerconfig

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/test/assertions"
	corev1 "k8s.io/api/core/v1"
)

type (
	ready              struct{}
	notReady           struct{}
	notReadyWithReason struct {
		expectedReason string
	}
)

var (
	_ assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig]           = (*ready)(nil)
	_ assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig]           = (*notReady)(nil)
	_ assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig]           = (*notReadyWithReason)(nil)
	_ assertions.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*ready)(nil)
	_ assertions.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*notReady)(nil)
	_ assertions.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*notReadyWithReason)(nil)
)

func (*ready) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	return condition.IsTrueWithReason(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, toolchainv1alpha1.SpaceProvisionerConfigValidReason)
}

func (*ready) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
	spc.Status.Conditions, _ = condition.AddOrUpdateStatusConditions(spc.Status.Conditions, toolchainv1alpha1.Condition{
		Type:   toolchainv1alpha1.ConditionReady,
		Status: corev1.ConditionTrue,
		Reason: toolchainv1alpha1.SpaceProvisionerConfigValidReason,
	})
	return spc
}

func Ready() assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &ready{}
}

func (*notReady) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	return condition.IsFalse(spc.Status.Conditions, toolchainv1alpha1.ConditionReady)
}

func (*notReady) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
	cnd, found := condition.FindConditionByType(spc.Status.Conditions, toolchainv1alpha1.ConditionReady)
	if !found {
		spc.Status.Conditions = condition.AddStatusConditions(spc.Status.Conditions, toolchainv1alpha1.Condition{
			Type:   toolchainv1alpha1.ConditionReady,
			Status: corev1.ConditionFalse,
		})
	} else {
		cnd.Status = corev1.ConditionFalse
		spc.Status.Conditions, _ = condition.AddOrUpdateStatusConditions(spc.Status.Conditions, cnd)
	}
	return spc
}

func NotReady() assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &notReady{}
}

func (p *notReadyWithReason) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	return condition.IsFalseWithReason(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, p.expectedReason)
}

func (p *notReadyWithReason) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
	cnd, found := condition.FindConditionByType(spc.Status.Conditions, toolchainv1alpha1.ConditionReady)
	if !found {
		spc.Status.Conditions = condition.AddStatusConditions(spc.Status.Conditions, toolchainv1alpha1.Condition{
			Type:   toolchainv1alpha1.ConditionReady,
			Status: corev1.ConditionFalse,
			Reason: p.expectedReason,
		})
	} else {
		cnd.Status = corev1.ConditionFalse
		cnd.Reason = p.expectedReason
		spc.Status.Conditions, _ = condition.AddOrUpdateStatusConditions(spc.Status.Conditions, cnd)
	}
	return spc
}

func NotReadyWithReason(reason string) assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &notReadyWithReason{expectedReason: reason}
}
