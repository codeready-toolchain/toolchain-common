package spaceprovisionerconfig

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/test/assertions"
)

type predicate func(*toolchainv1alpha1.SpaceProvisionerConfig) bool

var _ assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] = predicate(nil)

func (p predicate) Matches(obj *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	return p(obj)
}

func Ready() assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return predicate(func(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
		return condition.IsTrueWithReason(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, toolchainv1alpha1.SpaceProvisionerConfigValidReason)
	})
}

func NotReady() assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return predicate(func(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
		return condition.IsFalse(spc.Status.Conditions, toolchainv1alpha1.ConditionReady)
	})
}

func NotReadyWithReason(reason string) assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return predicate(func(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
		return condition.IsFalseWithReason(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, reason)
	})
}
