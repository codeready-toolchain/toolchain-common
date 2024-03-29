package states

import toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"

func ApprovedManually(userSignup *toolchainv1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, toolchainv1alpha1.UserSignupStateApproved)
}

// SetApprovedManually when a user was manually approved by an admin
// Note: when a user is manually approved, the `UserSignup.Spec.States` contains a single entry: `approved`
func SetApprovedManually(userSignup *toolchainv1alpha1.UserSignup, approved bool) {
	setState(userSignup, toolchainv1alpha1.UserSignupStateApproved, approved)
	if approved {
		setState(userSignup, toolchainv1alpha1.UserSignupStateVerificationRequired, false)
		setState(userSignup, toolchainv1alpha1.UserSignupStateDeactivating, false)
		setState(userSignup, toolchainv1alpha1.UserSignupStateDeactivated, false)
	}
}

func VerificationRequired(userSignup *toolchainv1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, toolchainv1alpha1.UserSignupStateVerificationRequired)
}

func SetVerificationRequired(userSignup *toolchainv1alpha1.UserSignup, val bool) {
	setState(userSignup, toolchainv1alpha1.UserSignupStateVerificationRequired, val)
}

func Deactivating(userSignup *toolchainv1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, toolchainv1alpha1.UserSignupStateDeactivating)
}

func SetDeactivating(userSignup *toolchainv1alpha1.UserSignup, val bool) {
	setState(userSignup, toolchainv1alpha1.UserSignupStateDeactivating, val)

	if val {
		setState(userSignup, toolchainv1alpha1.UserSignupStateDeactivated, false)
	}
}

func Deactivated(userSignup *toolchainv1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, toolchainv1alpha1.UserSignupStateDeactivated)
}

func SetDeactivated(userSignup *toolchainv1alpha1.UserSignup, deactivated bool) {
	setState(userSignup, toolchainv1alpha1.UserSignupStateDeactivated, deactivated)
	if deactivated {
		setState(userSignup, toolchainv1alpha1.UserSignupStateApproved, false)
		setState(userSignup, toolchainv1alpha1.UserSignupStateDeactivating, false)
	}
}

func setState(userSignup *toolchainv1alpha1.UserSignup, state toolchainv1alpha1.UserSignupState, val bool) {
	if val && !contains(userSignup.Spec.States, state) {
		userSignup.Spec.States = append(userSignup.Spec.States, state)
	}

	if !val && contains(userSignup.Spec.States, state) {
		userSignup.Spec.States = remove(userSignup.Spec.States, state)
	}
}

func contains(s []toolchainv1alpha1.UserSignupState, state toolchainv1alpha1.UserSignupState) bool {
	for _, a := range s {
		if a == state {
			return true
		}
	}
	return false
}

func remove(s []toolchainv1alpha1.UserSignupState, state toolchainv1alpha1.UserSignupState) []toolchainv1alpha1.UserSignupState {
	for i, v := range s {
		if v == state {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
