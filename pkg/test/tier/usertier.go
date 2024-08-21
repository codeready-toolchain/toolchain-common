package tier

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func WithDeactivationTimeoutDays(days int) Modifier {
	return func(userSignup *toolchainv1alpha1.UserTier) {
		userSignup.Spec.DeactivationTimeoutDays = days
	}
}

func WithName(name string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserTier) {
		userSignup.Name = name
	}
}

type Modifier func(tier *toolchainv1alpha1.UserTier)

func NewUserTier(modifiers ...Modifier) *toolchainv1alpha1.UserTier {
	t := &toolchainv1alpha1.UserTier{
		ObjectMeta: metav1.ObjectMeta{
			Name:              uuid.NewString(),
			Namespace:         test.HostOperatorNs,
			CreationTimestamp: metav1.Now(),
		},
	}
	for _, modify := range modifiers {
		modify(t)
	}
	return t
}
