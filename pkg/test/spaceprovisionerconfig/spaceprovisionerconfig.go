package spaceprovisionerconfig

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CreateOption func(*toolchainv1alpha1.SpaceProvisionerConfig)

func NewSpaceProvisionerConfig(name string, namespace string, opts ...CreateOption) *toolchainv1alpha1.SpaceProvisionerConfig {
	spc := &toolchainv1alpha1.SpaceProvisionerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	for _, apply := range opts {
		apply(spc)
	}

	return spc
}

func ReferencingToolchainCluster(name string) CreateOption {
	return func(spc *toolchainv1alpha1.SpaceProvisionerConfig) {
		spc.Spec.ToolchainCluster = name
	}
}

func Enabled(enabled bool) CreateOption {
	return func(spc *toolchainv1alpha1.SpaceProvisionerConfig) {
		spc.Spec.Enabled = enabled
	}
}

func WithReadyCondition() CreateOption {
	return func(spc *toolchainv1alpha1.SpaceProvisionerConfig) {
		spc.Status.Conditions, _ = condition.AddOrUpdateStatusConditions(spc.Status.Conditions, toolchainv1alpha1.Condition{
			Type:   toolchainv1alpha1.ConditionReady,
			Status: corev1.ConditionTrue,
			Reason: toolchainv1alpha1.SpaceProvisionerConfigValidReason,
		})
	}
}

func MaxNumberOfSpaces(number uint) CreateOption {
	return func(spc *toolchainv1alpha1.SpaceProvisionerConfig) {
		spc.Spec.CapacityThresholds.MaxNumberOfSpaces = number
	}
}

func MaxMemoryUtilizationPercent(number uint) CreateOption {
	return func(spc *toolchainv1alpha1.SpaceProvisionerConfig) {
		spc.Spec.CapacityThresholds.MaxMemoryUtilizationPercent = number
	}
}
