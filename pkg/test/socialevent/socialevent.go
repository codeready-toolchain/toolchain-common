package socialevent

import (
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewSocialEvent(namespace, name string, options ...Option) *toolchainv1alpha1.SocialEvent { // nolint:unparam
	event := &toolchainv1alpha1.SocialEvent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: toolchainv1alpha1.SocialEventSpec{
			UserTier:     "deactivate30",
			SpaceTier:    "base1ns",
			MaxAttendees: 10,
			StartTime:    metav1.NewTime(time.Now().Add(-1 * time.Hour)), // opened 1hr ago
			EndTime:      metav1.NewTime(time.Now().Add(1 * time.Hour)),  // closing in 1hr
		},
	}
	for _, apply := range options {
		apply(event)
	}
	return event
}

type Option func(*toolchainv1alpha1.SocialEvent)

func WithStartTime(start time.Time) Option {
	return func(event *toolchainv1alpha1.SocialEvent) {
		event.Spec.StartTime = metav1.NewTime(start)
	}
}

func WithEndTime(end time.Time) Option {
	return func(event *toolchainv1alpha1.SocialEvent) {
		event.Spec.EndTime = metav1.NewTime(end)
	}
}

func WithActivationCount(value int) Option {
	return func(event *toolchainv1alpha1.SocialEvent) {
		event.Status.ActivationCount = value
	}
}

func WithUserTier(tier string) Option {
	return func(event *toolchainv1alpha1.SocialEvent) {
		event.Spec.UserTier = tier
	}
}

func WithSpaceTier(tier string) Option {
	return func(event *toolchainv1alpha1.SocialEvent) {
		event.Spec.SpaceTier = tier
	}
}

func WithMaxAttendees(value int) Option {
	return func(event *toolchainv1alpha1.SocialEvent) {
		event.Spec.MaxAttendees = value
	}
}
