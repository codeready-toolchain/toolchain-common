package api

import (
	vmapiv1 "github.com/codeready-toolchain/toolchain-common/pkg/virtualmachine/api/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewVM(name, namespace string, options ...VMOption) *vmapiv1.VirtualMachine {
	vm := &vmapiv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: vmapiv1.VirtualMachineSpec{
			Template: &vmapiv1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: vmapiv1.VirtualMachineInstanceSpec{
					Domain: vmapiv1.DomainSpec{},
				},
			},
		},
	}

	for _, option := range options {
		option(vm)
	}

	return vm
}

type VMOption func(*vmapiv1.VirtualMachine)

func WithRequests(requests corev1.ResourceList) VMOption {
	return func(vm *vmapiv1.VirtualMachine) {
		vm.Spec.Template.Spec.Domain.Resources.Requests = requests
	}
}

func WithLimits(limits corev1.ResourceList) VMOption {
	return func(vm *vmapiv1.VirtualMachine) {
		vm.Spec.Template.Spec.Domain.Resources.Limits = limits
	}
}

func ResourceList(mem, cpu string) corev1.ResourceList {
	req := corev1.ResourceList{}
	if mem != "" {
		req["memory"] = resource.MustParse(mem)
	}
	if cpu != "" {
		req["cpu"] = resource.MustParse(cpu)
	}
	return req
}
