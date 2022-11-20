package spacebinding

import (
	"testing"

	commonsignup "github.com/codeready-toolchain/toolchain-common/pkg/test/usersignup"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	. "github.com/codeready-toolchain/toolchain-common/pkg/test/masteruserrecord"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewSpaceBinding(t *testing.T) {
	// given
	userSignup := commonsignup.NewUserSignup(commonsignup.WithName("johny"))
	space := newSpace(userSignup, test.MemberClusterName, "smith", "advanced")
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
}

func newSpace(userSignup *toolchainv1alpha1.UserSignup, targetCluster, compliantUserName, tier string) *toolchainv1alpha1.Space {
	labels := map[string]string{
		toolchainv1alpha1.SpaceCreatorLabelKey: userSignup.Name,
	}

	space := &toolchainv1alpha1.Space{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: userSignup.Namespace,
			Name:      compliantUserName,
			Labels:    labels,
		},
		Spec: toolchainv1alpha1.SpaceSpec{
			TargetCluster: targetCluster,
			TierName:      tier,
		},
	}
	return space
}
