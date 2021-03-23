package states

import "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"

func Deactivating(userSignup *v1alpha1.UserSignup) bool {
	return contains(userSignup.Spec.States, v1alpha1.UserSignupStateDeactivating)
}

func SetUserSignupState(userSignup *v1alpha1.UserSignup, state v1alpha1.UserSignupState) {
	if !contains(userSignup.Spec.States, state) {
		userSignup.Spec.States = append(userSignup.Spec.States, state)
	}
}

func UnsetUserSignupState(userSignup *v1alpha1.UserSignup, state v1alpha1.UserSignupState) {
	userSignup.Spec.States = remove(userSignup.Spec.States, state)
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
