package test

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HostOperatorConfigOptionFunc func(config *toolchainv1alpha1.HostOperatorConfig)

type HostOperatorConfigOption interface {
	Apply(config *toolchainv1alpha1.HostOperatorConfig)
}

type HostOperatorConfigOptionImpl struct {
	toApply []HostOperatorConfigOptionFunc
}

func (option *HostOperatorConfigOptionImpl) Apply(config *toolchainv1alpha1.HostOperatorConfig) {
	for _, apply := range option.toApply {
		apply(config)
	}
}

func (option *HostOperatorConfigOptionImpl) addFunction(funcToAdd HostOperatorConfigOptionFunc) {
	option.toApply = append(option.toApply, funcToAdd)
}

type AutomaticApprovalOption struct {
	*HostOperatorConfigOptionImpl
}

func AutomaticApproval() *AutomaticApprovalOption {
	o := &AutomaticApprovalOption{
		HostOperatorConfigOptionImpl: &HostOperatorConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.HostOperatorConfig) {
		config.Spec.AutomaticApproval = toolchainv1alpha1.AutomaticApproval{}
	})
	return o
}

func (o AutomaticApprovalOption) Enabled() AutomaticApprovalOption {
	o.addFunction(func(config *toolchainv1alpha1.HostOperatorConfig) {
		config.Spec.AutomaticApproval.Enabled = true
	})
	return o
}

func (o AutomaticApprovalOption) Disabled() AutomaticApprovalOption {
	o.addFunction(func(config *toolchainv1alpha1.HostOperatorConfig) {
		config.Spec.AutomaticApproval.Enabled = false
	})
	return o
}

type DeactivationOption struct {
	*HostOperatorConfigOptionImpl
}

func Deactivation() *DeactivationOption {
	o := &DeactivationOption{
		HostOperatorConfigOptionImpl: &HostOperatorConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.HostOperatorConfig) {
		config.Spec.Deactivation = toolchainv1alpha1.Deactivation{}
	})
	return o
}

func (o DeactivationOption) DeactivatingNotificationDays(days int) DeactivationOption {
	o.addFunction(func(config *toolchainv1alpha1.HostOperatorConfig) {
		config.Spec.Deactivation.DeactivatingNotificationDays = days
	})
	return o
}

type PerMemberClusterOption func(map[string]int)

func PerMemberCluster(name string, value int) PerMemberClusterOption {
	return func(clusters map[string]int) {
		clusters[name] = value
	}
}

func (o AutomaticApprovalOption) ResourceCapThreshold(defaultThreshold int, perMember ...PerMemberClusterOption) AutomaticApprovalOption {
	o.addFunction(func(config *toolchainv1alpha1.HostOperatorConfig) {
		config.Spec.AutomaticApproval.ResourceCapacityThreshold.DefaultThreshold = defaultThreshold
		config.Spec.AutomaticApproval.ResourceCapacityThreshold.SpecificPerMemberCluster = map[string]int{}
		for _, add := range perMember {
			add(config.Spec.AutomaticApproval.ResourceCapacityThreshold.SpecificPerMemberCluster)
		}
	})
	return o
}

func (o AutomaticApprovalOption) MaxUsersNumber(overall int, perMember ...PerMemberClusterOption) AutomaticApprovalOption {
	o.addFunction(func(config *toolchainv1alpha1.HostOperatorConfig) {
		config.Spec.AutomaticApproval.MaxNumberOfUsers.Overall = overall
		config.Spec.AutomaticApproval.MaxNumberOfUsers.SpecificPerMemberCluster = map[string]int{}
		for _, add := range perMember {
			add(config.Spec.AutomaticApproval.MaxNumberOfUsers.SpecificPerMemberCluster)
		}
	})
	return o
}

func NewHostOperatorConfig(options ...HostOperatorConfigOption) *toolchainv1alpha1.HostOperatorConfig {
	hostOperatorConfig := &toolchainv1alpha1.HostOperatorConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: HostOperatorNs,
			Name:      "config",
		},
	}
	for _, option := range options {
		option.Apply(hostOperatorConfig)
	}
	return hostOperatorConfig
}
