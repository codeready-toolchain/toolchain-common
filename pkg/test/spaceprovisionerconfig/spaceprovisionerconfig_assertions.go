package spaceprovisionerconfig

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
)

type (
	// readyWithStatusAndReason struct {
	// 	expectedStatus corev1.ConditionStatus
	// 	expectedReason *string
	// }

	consumedSpaceCount struct {
		expectedSpaceCount int
	}

	consumedMemoryUsage struct {
		expectedMemoryUsage map[string]int
	}

	unknownConsumedCapacity struct{}
)

var (
	// _ test.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*readyWithStatusAndReason)(nil)
	_ test.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*consumedSpaceCount)(nil)
	_ test.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*consumedMemoryUsage)(nil)
	_ test.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*unknownConsumedCapacity)(nil)

	// _ test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] = (*readyWithStatusAndReason)(nil)
	_ test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] = (*consumedSpaceCount)(nil)
	_ test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] = (*consumedMemoryUsage)(nil)
	_ test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] = (*unknownConsumedCapacity)(nil)

	conditionsAccessor = func(spc *toolchainv1alpha1.SpaceProvisionerConfig) *[]toolchainv1alpha1.Condition {
		return &spc.Status.Conditions
	}
)

//
// func (*readyWithStatusAndReason) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
// 	return condition.IsTrueWithReason(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, toolchainv1alpha1.SpaceProvisionerConfigValidReason)
// }
//
// func (r *readyWithStatusAndReason) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
// 	spc = spc.DeepCopyObject().(*toolchainv1alpha1.SpaceProvisionerConfig)
// 	cnd, found := condition.FindConditionByType(spc.Status.Conditions, toolchainv1alpha1.ConditionReady)
// 	if !found {
// 		spc.Status.Conditions = condition.AddStatusConditions(spc.Status.Conditions, toolchainv1alpha1.Condition{
// 			Type:   toolchainv1alpha1.ConditionReady,
// 			Status: corev1.ConditionFalse,
// 		})
// 	} else {
// 		cnd.Status = corev1.ConditionFalse
// 		spc.Status.Conditions, _ = condition.AddOrUpdateStatusConditions(spc.Status.Conditions, cnd)
// 	}
// 	return spc
// }

func (p *consumedSpaceCount) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	if spc.Status.ConsumedCapacity == nil {
		return false
	}
	return p.expectedSpaceCount == spc.Status.ConsumedCapacity.SpaceCount
}

func (p *consumedSpaceCount) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
	spc = spc.DeepCopyObject().(*toolchainv1alpha1.SpaceProvisionerConfig)
	if spc.Status.ConsumedCapacity == nil {
		spc.Status.ConsumedCapacity = &toolchainv1alpha1.ConsumedCapacity{}
	}
	spc.Status.ConsumedCapacity.SpaceCount = p.expectedSpaceCount
	return spc
}

func (p *consumedMemoryUsage) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	if spc.Status.ConsumedCapacity == nil {
		return false
	}
	if len(spc.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole) != len(p.expectedMemoryUsage) {
		return false
	}
	for k, v := range spc.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole {
		if p.expectedMemoryUsage[k] != v {
			return false
		}
	}
	return true
}

func (p *consumedMemoryUsage) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
	spc = spc.DeepCopyObject().(*toolchainv1alpha1.SpaceProvisionerConfig)
	if spc.Status.ConsumedCapacity == nil {
		spc.Status.ConsumedCapacity = &toolchainv1alpha1.ConsumedCapacity{}
	}
	spc.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole = p.expectedMemoryUsage
	return spc
}

func (p *unknownConsumedCapacity) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	return spc.Status.ConsumedCapacity == nil
}

func (p *unknownConsumedCapacity) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
	spc = spc.DeepCopyObject().(*toolchainv1alpha1.SpaceProvisionerConfig)
	spc.Status.ConsumedCapacity = nil
	return spc
}

func ReadyConditionThat(pred test.Predicate[[]toolchainv1alpha1.Condition]) test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return test.BridgeToConditions(func(spc *toolchainv1alpha1.SpaceProvisionerConfig) *[]toolchainv1alpha1.Condition {
		return &spc.Status.Conditions
	}, pred)
}

// func Ready() test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
// 	return &readyWithStatusAndReason{expectedStatus: corev1.ConditionTrue}
// }
//
// func NotReady() test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
// 	return &readyWithStatusAndReason{expectedStatus: corev1.ConditionFalse}
// }
//
// func NotReadyWithReason(reason string) test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
// 	return &readyWithStatusAndReason{expectedStatus: corev1.ConditionFalse, expectedReason: &reason}
// }
//
// func ReadyStatusAndReason(status corev1.ConditionStatus, reason string) test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
// 	return &readyWithStatusAndReason{expectedStatus: status, expectedReason: &reason}
// }

func ConsumedSpaceCount(value int) test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &consumedSpaceCount{expectedSpaceCount: value}
}

func ConsumedMemoryUsage(values map[string]int) test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &consumedMemoryUsage{expectedMemoryUsage: values}
}

func UnknownConsumedCapacity() test.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &unknownConsumedCapacity{}
}
