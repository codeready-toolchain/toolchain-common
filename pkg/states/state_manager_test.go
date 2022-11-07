package states

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestStateManager(t *testing.T) {

	u := &toolchainv1alpha1.UserSignup{}

	t.Run("test manually approved", func(t *testing.T) {

		SetManuallyApproved(u, true)

		require.True(t, ManuallyApproved(u))
		require.Len(t, u.Spec.States, 1)
		require.Equal(t, toolchainv1alpha1.UserSignupStateApproved, u.Spec.States[0])

		SetManuallyApproved(u, false)

		require.Empty(t, u.Spec.States)
		require.False(t, ManuallyApproved(u))

		SetDeactivated(u, true)
		SetVerificationRequired(u, true)
		SetManuallyApproved(u, true)

		// Setting approved should remove verification required
		require.False(t, VerificationRequired(u))

		// Setting approved should remove deactivated
		require.False(t, Deactivated(u))

		SetManuallyApproved(u, false)

		SetDeactivating(u, true)
		SetManuallyApproved(u, true)

		// Setting approved should remove deactivating
		require.False(t, Deactivating(u))
	})

	t.Run("test automatically approved", func(t *testing.T) {

		SetAutomaticallyApproved(u, true)

		require.True(t, AutomaticallyApproved(u))
		require.Empty(t, u.Spec.States)

		SetAutomaticallyApproved(u, false)
		require.Empty(t, u.Spec.States) // still empty

		SetDeactivated(u, true)
		SetVerificationRequired(u, true)
		SetAutomaticallyApproved(u, true)

		// Setting approved should remove verification required
		require.False(t, VerificationRequired(u))

		// Setting approved should remove deactivated
		require.False(t, Deactivated(u))

		SetAutomaticallyApproved(u, false)

		SetDeactivating(u, true)
		SetAutomaticallyApproved(u, true)

		// Setting approved should remove deactivating
		require.False(t, Deactivating(u))
	})

	t.Run("test verification required", func(t *testing.T) {
		SetAutomaticallyApproved(u, false)
		SetVerificationRequired(u, true)

		require.True(t, VerificationRequired(u))

		require.Len(t, u.Spec.States, 1)
		require.Equal(t, toolchainv1alpha1.UserSignupStateVerificationRequired, u.Spec.States[0])

		SetVerificationRequired(u, false)

		require.Empty(t, u.Spec.States)
		require.False(t, VerificationRequired(u))
	})

	t.Run("test deactivating", func(t *testing.T) {
		SetDeactivating(u, true)

		require.True(t, Deactivating(u))

		require.False(t, Deactivated(u))
		require.Len(t, u.Spec.States, 1)
		require.Equal(t, toolchainv1alpha1.UserSignupStateDeactivating, u.Spec.States[0])

		SetDeactivating(u, false)

		require.Empty(t, u.Spec.States)
		require.False(t, Deactivating(u))

		SetDeactivated(u, true)
		SetDeactivating(u, true)

		// Setting deactivating should also set deactivated to false
		require.False(t, Deactivated(u))
	})

	t.Run("test deactivated", func(t *testing.T) {
		SetDeactivated(u, true)

		require.True(t, Deactivated(u))
		require.Len(t, u.Spec.States, 1)
		require.Equal(t, toolchainv1alpha1.UserSignupStateDeactivated, u.Spec.States[0])

		SetDeactivated(u, false)
		require.Empty(t, u.Spec.States)

		SetDeactivating(u, true)
		SetAutomaticallyApproved(u, true)
		SetDeactivated(u, true)

		// Setting deactivated should also set approved and deactivating to false
		require.False(t, ManuallyApproved(u))
		require.False(t, Deactivating(u))
	})
}
