package nstemplateset

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterResourcesTemplateRef = "basic-clusterresources-abcde00"
	devTemplateRef              = "basic-dev-abcde11"
	codeTemplateRef             = "basic-code-abcde21"
)

func NewNSTemplateSet(name string, options ...Option) *toolchainv1alpha1.NSTemplateSet {
	nstmplSet := &toolchainv1alpha1.NSTemplateSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.MemberOperatorNs,
			Name:      name,
		},
		Spec: toolchainv1alpha1.NSTemplateSetSpec{
			TierName: "base1ns",
			ClusterResources: &toolchainv1alpha1.NSTemplateSetClusterResources{
				TemplateRef: clusterResourcesTemplateRef,
			},
			Namespaces: []toolchainv1alpha1.NSTemplateSetNamespace{
				{
					TemplateRef: devTemplateRef,
				},
				{
					TemplateRef: codeTemplateRef,
				},
			},
		},
		Status: toolchainv1alpha1.NSTemplateSetStatus{},
	}
	for _, apply := range options {
		apply(nstmplSet)
	}
	return nstmplSet
}

type Option func(*toolchainv1alpha1.NSTemplateSet)

func WithReferencesFor(nstemplateTier *toolchainv1alpha1.NSTemplateTier, opts ...TierOption) Option {
	return func(nstmplSet *toolchainv1alpha1.NSTemplateSet) {
		nstmplSet.Spec.TierName = nstemplateTier.Name

		// cluster resources
		if nstemplateTier.Spec.ClusterResources != nil {
			nstmplSet.Spec.ClusterResources = &toolchainv1alpha1.NSTemplateSetClusterResources{
				TemplateRef: nstemplateTier.Status.Revisions[nstemplateTier.Spec.ClusterResources.TemplateRef],
			}
		}

		// namespace resources
		if len(nstemplateTier.Spec.Namespaces) > 0 {
			nstmplSet.Spec.Namespaces = make([]toolchainv1alpha1.NSTemplateSetNamespace, len(nstemplateTier.Spec.Namespaces))
			for i, ns := range nstemplateTier.Spec.Namespaces {
				nstmplSet.Spec.Namespaces[i] = toolchainv1alpha1.NSTemplateSetNamespace{
					TemplateRef: nstemplateTier.Status.Revisions[ns.TemplateRef],
				}
			}
		}

		for _, apply := range opts {
			apply(nstemplateTier, nstmplSet)
		}
	}
}

type TierOption func(*toolchainv1alpha1.NSTemplateTier, *toolchainv1alpha1.NSTemplateSet)

func WithSpaceRole(role, username string) TierOption {
	return func(nstemplateTier *toolchainv1alpha1.NSTemplateTier, nstmplSet *toolchainv1alpha1.NSTemplateSet) {
		if tierSpaceRole, found := nstemplateTier.Spec.SpaceRoles[role]; found {
			// find the space role matching the templateref in the NSTemplateSet, and add the username
			for i := range nstmplSet.Spec.SpaceRoles {
				if nstmplSet.Spec.SpaceRoles[i].TemplateRef == nstemplateTier.Status.Revisions[tierSpaceRole.TemplateRef] {
					nstmplSet.Spec.SpaceRoles[i].Usernames = append(nstmplSet.Spec.SpaceRoles[i].Usernames, username)
					return
				}
			}
			// no entry for this space role yet, so let's add it
			nstmplSet.Spec.SpaceRoles = append(nstmplSet.Spec.SpaceRoles, toolchainv1alpha1.NSTemplateSetSpaceRole{
				TemplateRef: nstemplateTier.Status.Revisions[nstemplateTier.Spec.SpaceRoles[role].TemplateRef],
				Usernames:   []string{username},
			})
		}
	}
}

func WithReadyCondition() Option {
	return func(nstmplSet *toolchainv1alpha1.NSTemplateSet) {
		nstmplSet.Status.Conditions = []toolchainv1alpha1.Condition{
			{
				Type:   toolchainv1alpha1.ConditionReady,
				Status: corev1.ConditionTrue,
				Reason: toolchainv1alpha1.NSTemplateSetProvisionedReason,
			},
		}
	}
}

func WithNotReadyCondition(reason, message string) Option {
	return func(nstmplSet *toolchainv1alpha1.NSTemplateSet) {
		nstmplSet.Status.Conditions = []toolchainv1alpha1.Condition{
			{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  reason,
				Message: message,
			},
		}
	}
}

func WithDeletionTimestamp(ts time.Time) Option {
	return func(nstmplSet *toolchainv1alpha1.NSTemplateSet) {
		nstmplSet.DeletionTimestamp = &metav1.Time{Time: ts}
	}
}

func WithFinalizer() Option {
	return func(nstmplSet *toolchainv1alpha1.NSTemplateSet) {
		controllerutil.AddFinalizer(nstmplSet, toolchainv1alpha1.FinalizerName)
	}
}

func WithAnnotation(key, value string) Option {
	return func(nstmplSet *toolchainv1alpha1.NSTemplateSet) {
		if nstmplSet.ObjectMeta.Annotations == nil {
			nstmplSet.ObjectMeta.Annotations = map[string]string{}
		}
		nstmplSet.ObjectMeta.Annotations[key] = value

	}
}
