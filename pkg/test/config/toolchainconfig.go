package config

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ToolchainConfigOptionFunc func(config *toolchainv1alpha1.ToolchainConfig)

type ToolchainConfigOption interface {
	Apply(config *toolchainv1alpha1.ToolchainConfig)
}

type ToolchainConfigOptionImpl struct {
	toApply []ToolchainConfigOptionFunc
}

func (option *ToolchainConfigOptionImpl) Apply(config *toolchainv1alpha1.ToolchainConfig) {
	for _, apply := range option.toApply {
		apply(config)
	}
}

func (option *ToolchainConfigOptionImpl) addFunction(funcToAdd ToolchainConfigOptionFunc) {
	option.toApply = append(option.toApply, funcToAdd)
}

type AutomaticApprovalOption struct {
	*ToolchainConfigOptionImpl
}

func AutomaticApproval() *AutomaticApprovalOption {
	o := &AutomaticApprovalOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.AutomaticApproval = toolchainv1alpha1.AutomaticApprovalConfig{}
	})
	return o
}

func (o AutomaticApprovalOption) Enabled() AutomaticApprovalOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		val := true
		config.Spec.Host.AutomaticApproval.Enabled = &val
	})
	return o
}

func (o AutomaticApprovalOption) Disabled() AutomaticApprovalOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		val := false
		config.Spec.Host.AutomaticApproval.Enabled = &val
	})
	return o
}

type DeactivationOption struct {
	*ToolchainConfigOptionImpl
}

func Deactivation() *DeactivationOption {
	o := &DeactivationOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Deactivation = toolchainv1alpha1.DeactivationConfig{}
	})
	return o
}

func (o DeactivationOption) DeactivatingNotificationDays(days int) DeactivationOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Deactivation.DeactivatingNotificationDays = &days
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
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.AutomaticApproval.ResourceCapacityThreshold.DefaultThreshold = &defaultThreshold
		config.Spec.Host.AutomaticApproval.ResourceCapacityThreshold.SpecificPerMemberCluster = map[string]int{}
		for _, add := range perMember {
			add(config.Spec.Host.AutomaticApproval.ResourceCapacityThreshold.SpecificPerMemberCluster)
		}
	})
	return o
}

func (o AutomaticApprovalOption) MaxUsersNumber(overall int, perMember ...PerMemberClusterOption) AutomaticApprovalOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.AutomaticApproval.MaxNumberOfUsers.Overall = &overall
		config.Spec.Host.AutomaticApproval.MaxNumberOfUsers.SpecificPerMemberCluster = map[string]int{}
		for _, add := range perMember {
			add(config.Spec.Host.AutomaticApproval.MaxNumberOfUsers.SpecificPerMemberCluster)
		}
	})
	return o
}

type MembersOption struct {
	*ToolchainConfigOptionImpl
}

func Members() *MembersOption {
	o := &MembersOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	return o
}

func (o MembersOption) Default(memberConfigSpec toolchainv1alpha1.MemberOperatorConfigSpec) MembersOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Members.Default = memberConfigSpec
	})
	return o
}

func (o MembersOption) SpecificPerMemberCluster(clusterName string, memberConfigSpec toolchainv1alpha1.MemberOperatorConfigSpec) MembersOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		if config.Spec.Members.SpecificPerMemberCluster == nil {
			config.Spec.Members.SpecificPerMemberCluster = make(map[string]toolchainv1alpha1.MemberOperatorConfigSpec)
		}
		config.Spec.Members.SpecificPerMemberCluster[clusterName] = memberConfigSpec
	})
	return o
}

type MetricsOption struct {
	*ToolchainConfigOptionImpl
}

func Metrics() *MetricsOption {
	o := &MetricsOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Metrics = toolchainv1alpha1.MetricsConfig{}
	})
	return o
}

func (o MetricsOption) ForceSynchronization(force bool) MetricsOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Metrics.ForceSynchronization = &force
	})
	return o
}

func NewToolchainConfig(options ...ToolchainConfigOption) *toolchainv1alpha1.ToolchainConfig {
	toolchainConfig := &toolchainv1alpha1.ToolchainConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.HostOperatorNs,
			Name:      "config",
		},
	}
	for _, option := range options {
		option.Apply(toolchainConfig)
	}
	return toolchainConfig
}
