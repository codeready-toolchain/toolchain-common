package spacebindingrequest

import (
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Option func(spaceRequest *toolchainv1alpha1.SpaceBindingRequest)

func NewSpaceBindingRequest(name, namespace string, options ...Option) *toolchainv1alpha1.SpaceBindingRequest {
	spaceBindingRequest := &toolchainv1alpha1.SpaceBindingRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID(uuid.NewString()),
		},
	}
	for _, apply := range options {
		apply(spaceBindingRequest)
	}
	return spaceBindingRequest
}

func WithMUR(mur string) Option {
	return func(spaceBindingRequest *toolchainv1alpha1.SpaceBindingRequest) {
		spaceBindingRequest.Spec.MasterUserRecord = mur
	}
}

func WithSpaceRole(spaceRole string) Option {
	return func(spaceBindingRequest *toolchainv1alpha1.SpaceBindingRequest) {
		spaceBindingRequest.Spec.SpaceRole = spaceRole
	}
}

func WithLabel(key, value string) Option {
	return func(space *toolchainv1alpha1.SpaceBindingRequest) {
		if space.Labels == nil {
			space.Labels = map[string]string{}
		}
		space.Labels[key] = value
	}
}

func WithDeletionTimestamp() Option {
	return func(spaceBindingRequest *toolchainv1alpha1.SpaceBindingRequest) {
		now := metav1.NewTime(time.Now())
		spaceBindingRequest.DeletionTimestamp = &now
	}
}

func WithFinalizer() Option {
	return func(spaceBindingRequest *toolchainv1alpha1.SpaceBindingRequest) {
		controllerutil.AddFinalizer(spaceBindingRequest, toolchainv1alpha1.FinalizerName)
	}
}

func WithCondition(c toolchainv1alpha1.Condition) Option {
	return func(spaceBindingRequest *toolchainv1alpha1.SpaceBindingRequest) {
		spaceBindingRequest.Status.Conditions, _ = condition.AddOrUpdateStatusConditions(spaceBindingRequest.Status.Conditions, c)
	}
}
