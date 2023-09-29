package spacebinding

import (
	"testing"

	commonsignup "github.com/codeready-toolchain/toolchain-common/pkg/test/usersignup"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	. "github.com/codeready-toolchain/toolchain-common/pkg/test/masteruserrecord"
	spacetest "github.com/codeready-toolchain/toolchain-common/pkg/test/space"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSpaceBinding(t *testing.T) {
	// given
	userSignup := commonsignup.NewUserSignup(commonsignup.WithName("johny"))
	space := spacetest.NewSpace(test.HostOperatorNs, "smith",
		spacetest.WithTierName("advanced"),
		spacetest.WithSpecTargetCluster(test.MemberClusterName),
		spacetest.WithLabel(toolchainv1alpha1.SpaceCreatorLabelKey, userSignup.Name),
	)
	mur := NewMasterUserRecord(t, "johny", TargetCluster(test.MemberClusterName), TierName("deactivate90"))

	// when
	actualSpaceBinding := NewSpaceBinding(mur, space, userSignup.Name)

	// then
	assert.Equal(t, "johny", actualSpaceBinding.Spec.MasterUserRecord)
	assert.Equal(t, "smith", actualSpaceBinding.Spec.Space)
	assert.Equal(t, "admin", actualSpaceBinding.Spec.SpaceRole)

	require.NotNil(t, actualSpaceBinding.Labels)
	assert.Equal(t, userSignup.Name, actualSpaceBinding.Labels[toolchainv1alpha1.SpaceCreatorLabelKey])
	assert.Equal(t, "johny", actualSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingMasterUserRecordLabelKey])
	assert.Equal(t, "smith", actualSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingSpaceLabelKey])

	t.Run("with role", func(t *testing.T) {
		// when
		actualSpaceBinding := NewSpaceBinding(mur, space, userSignup.Name, WithRole("custom-role"))

		// then
		assert.Equal(t, "custom-role", actualSpaceBinding.Spec.SpaceRole)
	})
}
