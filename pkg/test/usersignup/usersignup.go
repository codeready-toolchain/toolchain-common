package usersignup

import (
	"strconv"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/hash"
	"github.com/codeready-toolchain/toolchain-common/pkg/states"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/codeready-toolchain/toolchain-common/pkg/usersignup"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func WithTargetCluster(targetCluster string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Spec.TargetCluster = targetCluster
	}
}

func WithOriginalSub(originalSub string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Spec.IdentityClaims.OriginalSub = originalSub
	}
}

func WithUserID(userID string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Spec.IdentityClaims.UserID = userID
	}
}

func WithAccountID(accountID string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Spec.IdentityClaims.AccountID = accountID
	}
}

// ApprovedManuallyAgo sets the UserSignup state to `approved` and adds a status condition
func ApprovedManuallyAgo(before time.Duration) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetApprovedManually(userSignup, true)
		userSignup.Status.Conditions = condition.AddStatusConditions(userSignup.Status.Conditions,
			toolchainv1alpha1.Condition{
				Type:               toolchainv1alpha1.UserSignupApproved,
				Status:             corev1.ConditionTrue,
				Reason:             toolchainv1alpha1.UserSignupApprovedByAdminReason,
				LastTransitionTime: metav1.Time{Time: time.Now().Add(-before)},
			})
	}
}

// ApprovedAutomaticallyAgo sets the UserSignup state to `approved` and adds a status condition
func ApprovedAutomaticallyAgo(before time.Duration) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetApprovedManually(userSignup, true)
		userSignup.Status.Conditions = condition.AddStatusConditions(userSignup.Status.Conditions,
			toolchainv1alpha1.Condition{
				Type:               toolchainv1alpha1.UserSignupApproved,
				Status:             corev1.ConditionTrue,
				Reason:             toolchainv1alpha1.UserSignupApprovedAutomaticallyReason,
				LastTransitionTime: metav1.Time{Time: time.Now().Add(-before)},
			})
	}
}

func Deactivated() Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetDeactivated(userSignup, true)
	}
}

func VerificationRequired() Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetVerificationRequired(userSignup, true)
	}
}

// ApprovedManually sets the UserSignup states to [`approved`]
func ApprovedManually() Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetApprovedManually(userSignup, true)
	}
}

func DeactivatedAgo(before time.Duration) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetDeactivated(userSignup, true)

		deactivatedCondition := toolchainv1alpha1.Condition{
			Type:               toolchainv1alpha1.UserSignupComplete,
			Status:             corev1.ConditionTrue,
			Reason:             toolchainv1alpha1.UserSignupUserDeactivatedReason,
			LastTransitionTime: metav1.Time{Time: time.Now().Add(-before)},
		}

		userSignup.Status.Conditions = condition.AddStatusConditions(userSignup.Status.Conditions, deactivatedCondition)
	}
}

func BannedAgo(before time.Duration) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Status.Conditions = condition.AddStatusConditions(userSignup.Status.Conditions,
			toolchainv1alpha1.Condition{
				Type:               toolchainv1alpha1.UserSignupComplete,
				Status:             corev1.ConditionTrue,
				Reason:             toolchainv1alpha1.UserSignupUserBannedReason,
				LastTransitionTime: metav1.Time{Time: time.Now().Add(-before)},
			})
	}
}

func VerificationRequiredAgo(before time.Duration) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		states.SetVerificationRequired(userSignup, true)

		verificationRequired := toolchainv1alpha1.Condition{
			Type:               toolchainv1alpha1.UserSignupComplete,
			Status:             corev1.ConditionFalse,
			Reason:             toolchainv1alpha1.UserSignupVerificationRequiredReason,
			LastTransitionTime: metav1.Time{Time: time.Now().Add(-before)},
		}

		userSignup.Status.Conditions = condition.AddStatusConditions(userSignup.Status.Conditions, verificationRequired)

	}
}

func WithUsername(username string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Spec.IdentityClaims.PreferredUsername = username
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
		emailHash := hash.EncodeString(email)
		userSignup.ObjectMeta.Labels[toolchainv1alpha1.UserSignupUserEmailHashLabelKey] = emailHash
		userSignup.Spec.IdentityClaims.Email = email
	}
}

func SignupComplete(reason string) Modifier {
	return func(userSignup *toolchainv1alpha1.UserSignup) {
		userSignup.Status.Conditions = condition.AddStatusConditions(userSignup.Status.Conditions,
			toolchainv1alpha1.Condition{
				Type:   toolchainv1alpha1.UserSignupComplete,
				Status: corev1.ConditionTrue,
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
		userSignup.Spec.IdentityClaims.PreferredUsername = name
	}
}

type Modifier func(*toolchainv1alpha1.UserSignup)

func NewUserSignup(modifiers ...Modifier) *toolchainv1alpha1.UserSignup {
	meta := NewUserSignupObjectMeta("", "foo@redhat.com")
	signup := &toolchainv1alpha1.UserSignup{
		ObjectMeta: meta,
		Spec: toolchainv1alpha1.UserSignupSpec{
			IdentityClaims: toolchainv1alpha1.IdentityClaimsEmbedded{
				PropagatedClaims: toolchainv1alpha1.PropagatedClaims{
					Sub:         "UserID123",
					UserID:      "0192837465",
					AccountID:   "5647382910",
					OriginalSub: "original-sub-value",
					Email:       "foo@redhat.com",
				},
				PreferredUsername: meta.Name,
				GivenName:         "Foo",
				FamilyName:        "Bar",
				Company:           "Red Hat",
			},
		},
	}
	for _, modify := range modifiers {
		modify(signup)
	}
	return signup
}

func NewUserSignupObjectMeta(name, email string) metav1.ObjectMeta {
	if name == "" {
		name = uuid.NewString()
		// limit to maxLength
		name = usersignup.TransformUsername(name, []string{"openshift", "kube", "default", "redhat", "sandbox"}, []string{"admin"})
	}
	emailHash := hash.EncodeString(email)

	return metav1.ObjectMeta{
		Name:      name,
		Namespace: test.HostOperatorNs,
		Labels: map[string]string{
			toolchainv1alpha1.UserSignupUserEmailHashLabelKey: emailHash,
		},
		Annotations:       make(map[string]string),
		CreationTimestamp: metav1.Now(),
	}
}
