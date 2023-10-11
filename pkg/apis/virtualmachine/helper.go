package virtualmachine

import (
	corev1 "k8s.io/api/core/v1"
)

type VMOption func(*VirtualMachine)

func WithRequests(requests corev1.ResourceList) VMOption {
	return func(vm *VirtualMachine) {
		vm.Spec.Template.Spec.Domain.Resources.Requests = requests
	}
}

func WithLimits(limits corev1.ResourceList) VMOption {
	return func(vm *VirtualMachine) {
		vm.Spec.Template.Spec.Domain.Resources.Limits = limits
	}
}
