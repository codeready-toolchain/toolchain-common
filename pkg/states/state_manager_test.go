package states

import (
	"github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStateManager(t *testing.T) {

	u := &v1alpha1.UserSignup{}

	t.Run("test set UserSignup state", func(t *testing.T) {

		SetUserSignupState(u, v1alpha1.UserSignupStateDeactivating)

		require.Len(t, u.Spec.States, 1)
		require.Equal(t, v1alpha1.UserSignupStateDeactivating, u.Spec.States[0])
	})

	t.Run("test unset UserSignup state", func(t *testing.T) {

		// Try to unset a state that isn't set
		UnsetUserSignupState(u, v1alpha1.UserSignupStateBanned)

		// Ensure the existing states haven't changed
		require.Len(t, u.Spec.States, 1)

		UnsetUserSignupState(u, v1alpha1.UserSignupStateDeactivating)

		require.Len(t, u.Spec.States, 0)
	})

	t.Run("test Deactivating function", func(t *testing.T) {
		require.False(t, Deactivating(u))

		SetUserSignupState(u, v1alpha1.UserSignupStateDeactivating)

		require.True(t, Deactivating(u))
	})
}
