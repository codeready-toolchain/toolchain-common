package states

import (
	"github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStateManager(t *testing.T) {

	u := &v1alpha1.UserSignup{}

	t.Run("test approved", func(t *testing.T) {

		SetApproved(u, true)

		require.True(t, Approved(u))
		require.Len(t, u.Spec.States, 1)
		require.Equal(t, v1alpha1.UserSignupStateApproved, u.Spec.States[0])

		SetApproved(u, false)

		require.Len(t, u.Spec.States, 0)
		require.False(t, Approved(u))

		SetVerificationRequired(u, true)
		SetApproved(u, true)

		// Setting approved should remove verification required
		require.False(t, VerificationRequired(u))
	})

	t.Run("test verification required", func(t *testing.T) {
		SetApproved(u, false)
		SetVerificationRequired(u, true)

		require.True(t, VerificationRequired(u))

		require.Len(t, u.Spec.States, 1)
		require.Equal(t, v1alpha1.UserSignupStateVerificationRequired, u.Spec.States[0])

		SetVerificationRequired(u, false)

		require.Len(t, u.Spec.States, 0)
		require.False(t, VerificationRequired(u))
	})

	t.Run("test deactivating", func(t *testing.T) {
		SetDeactivating(u, true)

		require.True(t, Deactivating(u))

		require.False(t, Deactivated(u))
		require.Len(t, u.Spec.States, 1)
		require.Equal(t, v1alpha1.UserSignupStateDeactivating, u.Spec.States[0])

		SetDeactivating(u, false)

		require.Len(t, u.Spec.States, 0)
		require.False(t, Deactivating(u))
	})

	t.Run("test deactivated", func(t *testing.T) {
		SetDeactivated(u, true)

		require.True(t, Deactivated(u))
		require.Len(t, u.Spec.States, 1)
		require.Equal(t, v1alpha1.UserSignupStateDeactivated, u.Spec.States[0])

		SetDeactivated(u, false)
		require.Len(t, u.Spec.States, 0)

		SetApproved(u, true)
		SetDeactivated(u, true)

		// Setting deactivated should also set approved to false
		require.False(t, Approved(u))
	})

	t.Run("test active", func(t *testing.T) {
		u = &v1alpha1.UserSignup{}
		// Should not be active by default
		require.False(t, Active(u))

		SetApproved(u, true)
		// Should be active when approved
		require.True(t, Active(u))

		SetVerificationRequired(u, true)
		// Should not be active when verification is required
		require.False(t, Active(u))

		SetDeactivated(u, true)
		// Should not be active when deactivated
		require.False(t, Active(u))
		require.False(t, Approved(u))

		// Reapprove
		SetApproved(u, true)

		SetVerificationRequired(u, false)
		// Should still not be active
		require.False(t, Active(u))

		SetDeactivated(u, false)
		// Should be active when verification not required and not deactivated
		require.True(t, Active(u))

		SetDeactivating(u, true)
		// Should be active when deactivating
		require.True(t, Active(u))
	})
}
