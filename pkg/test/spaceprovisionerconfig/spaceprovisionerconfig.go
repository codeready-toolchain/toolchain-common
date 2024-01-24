package spaceprovisionerconfig

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewSpaceProvisionerConfig(name string, namespace string, referencedToolchainCluster string) *toolchainv1alpha1.SpaceProvisionerConfig {
	return &toolchainv1alpha1.SpaceProvisionerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: toolchainv1alpha1.SpaceProvisionerConfigSpec{
			ToolchainCluster: referencedToolchainCluster,
		},
	}
}
