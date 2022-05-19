package usersignup

import (
	"crypto/md5" // nolint:gosec
	"encoding/hex"
	"strconv"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/states"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/gofrs/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func WithTargetCluster(targetCluster string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Spec.TargetCluster = targetCluster
	}
}

func WithOriginalSub(originalSub string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Spec.OriginalSub = originalSub
	}
}

func Approved() Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetApproved(userSignup, true)
	}
}

func ApprovedAutomatically(before time.Duration) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetApproved(userSignup, true)
		userSignup.Status.Conditions = condition.AddStatusConditions(userSignup.Status.Conditions,
			toolchainv1alpha1.Condition{
				Type:               toolchainv1alpha1.UserSignupApproved,
				Status:             v1.ConditionTrue,
				Reason:             "ApprovedAutomatically",
				LastTransitionTime: metav1.Time{Time: time.Now().Add(-before)},
			})
	}
}

func Deactivated() Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetDeactivated(userSignup, true)
	}
}

func DeactivatedWithLastTransitionTime(before time.Duration) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetDeactivated(userSignup, true)

		deactivatedCondition := toolchainv1alpha1.Condition{
			Type:               toolchainv1alpha1.UserSignupComplete,
			Status:             v1.ConditionTrue,
			Reason:             toolchainv1alpha1.UserSignupUserDeactivatedReason,
			LastTransitionTime: metav1.Time{Time: time.Now().Add(-before)},
		}

		userSignup.Status.Conditions = condition.AddStatusConditions(userSignup.Status.Conditions, deactivatedCondition)
	}
}

func VerificationRequired(before time.Duration) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetVerificationRequired(userSignup, true)

		verificationRequired := toolchainv1alpha1.Condition{
			Type:               toolchainv1alpha1.UserSignupComplete,
			Status:             v1.ConditionFalse,
			Reason:             toolchainv1alpha1.UserSignupVerificationRequiredReason,
			LastTransitionTime: metav1.Time{Time: time.Now().Add(-before)},
		}

		userSignup.Status.Conditions = condition.AddStatusConditions(userSignup.Status.Conditions, verificationRequired)

	}
}

func WithUsername(username string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Spec.Username = username
	}
}

func WithLabel(key, value string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Labels[key] = value
	}
}

func WithStateLabel(stateValue string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Labels[toolchainv1alpha1.UserSignupStateLabelKey] = stateValue
	}
}

func WithEmail(email string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		md5hash := md5.New() // nolint:gosec
		// Ignore the error, as this implementation cannot return one
		_, _ = md5hash.Write([]byte(email))
		emailHash := hex.EncodeToString(md5hash.Sum(nil))
		userSignup.ObjectMeta.Labels[toolchainv1alpha1.UserSignupUserEmailHashLabelKey] = emailHash
		userSignup.ObjectMeta.Annotations[toolchainv1alpha1.UserSignupUserEmailAnnotationKey] = email
	}
}

func SignupComplete(reason string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Status.Conditions = condition.AddStatusConditions(userSignup.Status.Conditions,
			toolchainv1alpha1.Condition{
				Type:   toolchainv1alpha1.UserSignupComplete,
				Status: v1.ConditionTrue,
				Reason: reason,
			})
	}
}

func CreatedBefore(before time.Duration) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now().Add(-before)}
	}
}

func BeingDeleted() Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	}
}

func WithActivations(value string) Modifier {
	return WithAnnotation(toolchainv1alpha1.UserSignupActivationCounterAnnotationKey, value)
}

func WithVerificationAttempts(value int) Modifier {
	return WithAnnotation(toolchainv1alpha1.UserVerificationAttemptsAnnotationKey, strconv.Itoa(value))
}

func WithAnnotation(key, value string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Annotations[key] = value
	}
}

func WithoutAnnotation(key string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		delete(userSignup.Annotations, key)
	}
}

func WithoutAnnotations() Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Annotations = map[string]string{}
	}
}

func WithName(name string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Name = name
		userSignup.Spec.Username = name
	}
}

type Modifier func(*toolchainv1alpha1.UserSignup)

func NewUserSignup(modifiers ...Modifier) *toolchainv1alpha1.UserSignup {
	meta := NewUserSignupObjectMeta("", "foo@redhat.com")
	signup := &toolchainv1alpha1.UserSignup{
		ObjectMeta: meta,
		Spec: toolchainv1alpha1.UserSignupSpec{
			Userid:   "UserID123",
			Username: meta.Name,
		},
	}
	for _, modify := range modifiers {
		modify(signup)
	}
	return signup
}

func NewUserSignupObjectMeta(name, email string) metav1.ObjectMeta {
	if name == "" {
		name = uuid.Must(uuid.NewV4()).String()
	}

	md5hash := md5.New() // nolint:gosec
	// Ignore the error, as this implementation cannot return one
	_, _ = md5hash.Write([]byte(email))
	emailHash := hex.EncodeToString(md5hash.Sum(nil))

	return metav1.ObjectMeta{
		Name:      name,
		Namespace: test.HostOperatorNs,
		Annotations: map[string]string{
			toolchainv1alpha1.UserSignupUserEmailAnnotationKey: email,
		},
		Labels: map[string]string{
			toolchainv1alpha1.UserSignupUserEmailHashLabelKey: emailHash,
		},
		CreationTimestamp: metav1.Now(),
	}
}
