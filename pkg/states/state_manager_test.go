package states

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestStateManager(t *testing.T) {

	t.Run("test manually approved", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}

			// when
			SetApprovedManually(u, true)

			// then
			require.True(t, ApprovedManually(u))
			require.Len(t, u.Spec.States, 1)
			require.Equal(t, toolchainv1alpha1.UserSignupStateApproved, u.Spec.States[0])
		})
		t.Run("false", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}

			// when
			SetApprovedManually(u, false)

			// then
			require.Empty(t, u.Spec.States)
			require.False(t, ApprovedManually(u))
		})

		t.Run("true with verification required and deactivated states", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}
			SetDeactivated(u, true)
			SetVerificationRequired(u, true)

			// when
			SetApprovedManually(u, true)

			// then
			// Setting approved should remove verification required
			require.False(t, VerificationRequired(u))

			// Setting approved should remove deactivated
			require.False(t, Deactivated(u))
		})

		t.Run("true with deactivating state", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}
			SetDeactivating(u, true)

			// when
			SetApprovedManually(u, true)

			// then
			// Setting approved should remove deactivating
			require.False(t, Deactivating(u))
		})

		t.Run("true with rejected state", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}
			SetRejected(u, true)

			// when
			SetApprovedManually(u, true)

			// then
			// Setting approved should remove rejected
			require.False(t, Rejected(u))
		})

		t.Run("true with no-provisioning state", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}
			SetNoProvisioning(u, true)

			// when
			SetApprovedManually(u, true)

			// then
			require.False(t, NoProvisioning(u))
		})
	})

	t.Run("test verification required", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}
			SetApprovedManually(u, false)

			// when
			SetVerificationRequired(u, true)

			// then
			require.True(t, VerificationRequired(u))
			require.Len(t, u.Spec.States, 1)
			require.Equal(t, toolchainv1alpha1.UserSignupStateVerificationRequired, u.Spec.States[0])
		})

		t.Run("true", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}

			// when
			SetVerificationRequired(u, false)

			// then
			require.Empty(t, u.Spec.States)
			require.False(t, VerificationRequired(u))
		})
	})

	t.Run("test deactivating", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}
			SetDeactivated(u, true)

			// when
			SetDeactivating(u, true)

			// then
			require.True(t, Deactivating(u))
			require.False(t, Deactivated(u))
			require.Len(t, u.Spec.States, 1)
			require.Equal(t, toolchainv1alpha1.UserSignupStateDeactivating, u.Spec.States[0])
		})

		t.Run("false", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}

			// when
			SetDeactivating(u, false)

			// then
			require.Empty(t, u.Spec.States)
			require.False(t, Deactivating(u))
		})
	})

	t.Run("test deactivated", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}

			// when
			SetDeactivated(u, true)

			// then
			require.True(t, Deactivated(u))
			require.Len(t, u.Spec.States, 1)
			require.Equal(t, toolchainv1alpha1.UserSignupStateDeactivated, u.Spec.States[0])
		})

		t.Run("false", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}

			// when
			SetDeactivated(u, false)

			// then
			require.Empty(t, u.Spec.States)
		})

		t.Run("true with existing states", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}
			SetDeactivating(u, true)
			SetApprovedManually(u, true)

			// when
			SetDeactivated(u, true)

			// then
			require.False(t, ApprovedManually(u))
			require.False(t, Deactivating(u))
		})
	})

	t.Run("test no provisioning", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}
			SetDeactivated(u, false)
			// when
			SetNoProvisioning(u, true)

			// then
			require.True(t, NoProvisioning(u))
			require.Len(t, u.Spec.States, 1)
			require.Equal(t, toolchainv1alpha1.UserSignupStateNoProvisioning, u.Spec.States[0])
		})

		t.Run("false", func(t *testing.T) {
			// given
			u := &toolchainv1alpha1.UserSignup{}

			// when
			SetNoProvisioning(u, false)

			// then
			require.Empty(t, u.Spec.States)
			require.False(t, NoProvisioning(u))
		})
	})
}
