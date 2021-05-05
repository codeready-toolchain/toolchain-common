package states

import "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"

func Active(userSignup *v1alpha1.UserSignup) bool {
	return Approved(userSignup) &&
		!VerificationRequired(userSignup) &&
		!Deactivated(userSignup)
}

func Approved(userSignup *v1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, v1alpha1.UserSignupStateApproved)
}

func SetApproved(userSignup *v1alpha1.UserSignup, val bool) {
	setState(userSignup, v1alpha1.UserSignupStateApproved, val)

	if val {
		setState(userSignup, v1alpha1.UserSignupStateVerificationRequired, false)
		setState(userSignup, v1alpha1.UserSignupStateDeactivating, false)
		setState(userSignup, v1alpha1.UserSignupStateDeactivated, false)
	}
}

func VerificationRequired(userSignup *v1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, v1alpha1.UserSignupStateVerificationRequired)
}

func SetVerificationRequired(userSignup *v1alpha1.UserSignup, val bool) {
	setState(userSignup, v1alpha1.UserSignupStateVerificationRequired, val)
}

func Deactivating(userSignup *v1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivating)
}

func SetDeactivating(userSignup *v1alpha1.UserSignup, val bool) {
	setState(userSignup, v1alpha1.UserSignupStateDeactivating, val)

	if val {
		setState(userSignup, v1alpha1.UserSignupStateDeactivated, false)
	}
}

func Deactivated(userSignup *v1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivated)
}

func SetDeactivated(userSignup *v1alpha1.UserSignup, val bool) {
	setState(userSignup, v1alpha1.UserSignupStateDeactivated, val)

	if val {
		setState(userSignup, v1alpha1.UserSignupStateApproved, false)
		setState(userSignup, v1alpha1.UserSignupStateDeactivating, false)
	}
}

func setState(userSignup *v1alpha1.UserSignup, state v1alpha1.UserSignupState, val bool) {
	if val && !contains(userSignup.Spec.States, state) {
		userSignup.Spec.States = append(userSignup.Spec.States, state)
	}

	if !val && contains(userSignup.Spec.States, state) {
		userSignup.Spec.States = remove(userSignup.Spec.States, state)
	}
}

func contains(s []v1alpha1.UserSignupState, state v1alpha1.UserSignupState) bool {
	for _, a := range s {
		if a == state {
			return true
		}
	}
	return false
}

func remove(s []v1alpha1.UserSignupState, state v1alpha1.UserSignupState) []v1alpha1.UserSignupState {
	for i, v := range s {
		if v == state {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
