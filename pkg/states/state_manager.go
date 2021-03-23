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
	if val && !contains(userSignup.Spec.States, v1alpha1.UserSignupStateApproved) {
		userSignup.Spec.States = append(userSignup.Spec.States, v1alpha1.UserSignupStateApproved)
	}

	if !val && contains(userSignup.Spec.States, v1alpha1.UserSignupStateApproved) {
		userSignup.Spec.States = remove(userSignup.Spec.States, v1alpha1.UserSignupStateApproved)
	}
}

func VerificationRequired(userSignup *v1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, v1alpha1.UserSignupStateVerificationRequired)
}

func SetVerificationRequired(userSignup *v1alpha1.UserSignup, val bool) {
	if val && !contains(userSignup.Spec.States, v1alpha1.UserSignupStateVerificationRequired) {
		userSignup.Spec.States = append(userSignup.Spec.States, v1alpha1.UserSignupStateVerificationRequired)
	}

	if !val && contains(userSignup.Spec.States, v1alpha1.UserSignupStateVerificationRequired) {
		userSignup.Spec.States = remove(userSignup.Spec.States, v1alpha1.UserSignupStateVerificationRequired)
	}
}

func Deactivating(userSignup *v1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivating)
}

func SetDeactivating(userSignup *v1alpha1.UserSignup, val bool) {
	if val && !contains(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivating) {
		userSignup.Spec.States = append(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivating)
	}

	if !val && contains(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivating) {
		userSignup.Spec.States = remove(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivating)
	}
}

func Deactivated(userSignup *v1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivated)
}

func SetDeactivated(userSignup *v1alpha1.UserSignup, val bool) {
	if val && !contains(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivated) {
		userSignup.Spec.States = append(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivated)
	}

	if !val && contains(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivated) {
		userSignup.Spec.States = remove(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivated)
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
