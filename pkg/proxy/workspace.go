package proxy

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkspaceOption func(*toolchainv1alpha1.Workspace)

func NewWorkspace(name string, options ...WorkspaceOption) *toolchainv1alpha1.Workspace {
	workspace := &toolchainv1alpha1.Workspace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workspace",
			APIVersion: "toolchain.dev.openshift.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	for _, option := range options {
		option(workspace)
	}
	return workspace
}

func WithNamespaces(namespaces []toolchainv1alpha1.WorkspaceNamespace) WorkspaceOption {
	return func(workspace *toolchainv1alpha1.Workspace) {
		workspace.Status.Namespaces = namespaces
	}
}

func WithOwner(owner string) WorkspaceOption {
	return func(workspace *toolchainv1alpha1.Workspace) {
		workspace.Status.Owner = owner
	}
}

func WithRole(role string) WorkspaceOption {
	return func(workspace *toolchainv1alpha1.Workspace) {
		workspace.Status.Role = role
	}
}
