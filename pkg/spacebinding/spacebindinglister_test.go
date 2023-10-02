package spacebinding_test

import (
	"fmt"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/spacebinding"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	. "github.com/codeready-toolchain/toolchain-common/pkg/test/masteruserrecord"
	spacetest "github.com/codeready-toolchain/toolchain-common/pkg/test/space"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSpaceBindingLister(t *testing.T) {

	t.Run("recursive list for space", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			// given
			space := spacetest.NewSpace(test.HostOperatorNs, "smith",
				spacetest.WithTierName("advanced"),
				spacetest.WithSpecTargetCluster(test.MemberClusterName),
				spacetest.WithLabel(toolchainv1alpha1.SpaceCreatorLabelKey, "johnny"),
			)
			subSpace := spacetest.NewSpace(test.HostOperatorNs, "smith-sub",
				spacetest.WithTierName("advanced"),
				spacetest.WithSpecTargetCluster(test.MemberClusterName),
				spacetest.WithLabel(toolchainv1alpha1.SpaceCreatorLabelKey, "johnny"),
				spacetest.WithSpecParentSpace(space.GetName()),
			)
			mur := NewMasterUserRecord(t, "johnny", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
			listSpaceBindingsFunc := func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(mur, space, "johnny", spacebinding.WithRole("admin")),
				}, nil
			}
			getSpaceFunc := func(spaceName string) (*toolchainv1alpha1.Space, error) {
				return space, nil
			}

			// when
			spaceBindingLister := spacebinding.NewLister(listSpaceBindingsFunc, getSpaceFunc)

			// then
			// listing the spacebindings for the sub-space should return the spacebinding from parent-space
			spaceBindings, err := spaceBindingLister.ListForSpace(subSpace, []toolchainv1alpha1.SpaceBinding{})
			assert.NoError(t, err)
			assert.Len(t, spaceBindings, 1, "invalid number of spacebidings")

			actualSpaceBinding := spaceBindings[0]
			assert.Equal(t, "johnny", actualSpaceBinding.Spec.MasterUserRecord)
			assert.Equal(t, "smith", actualSpaceBinding.Spec.Space)
			assert.Equal(t, "admin", actualSpaceBinding.Spec.SpaceRole)

			require.NotNil(t, actualSpaceBinding.Labels)
			assert.Equal(t, "johnny", actualSpaceBinding.Labels[toolchainv1alpha1.SpaceCreatorLabelKey])
			assert.Equal(t, "johnny", actualSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingMasterUserRecordLabelKey])
			assert.Equal(t, "smith", actualSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingSpaceLabelKey])
		})

		t.Run("error listing spacebindings", func(t *testing.T) {
			// given
			myspace := spacetest.NewSpace(test.HostOperatorNs, "myspace")
			listSpaceBindingsFunc := func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
				return nil, fmt.Errorf("error listing spacebindings")
			}
			getSpaceFunc := func(spaceName string) (*toolchainv1alpha1.Space, error) {
				return myspace, nil
			}

			// when
			spaceBindingLister := spacebinding.NewLister(listSpaceBindingsFunc, getSpaceFunc)

			// then
			spaceBindings, err := spaceBindingLister.ListForSpace(myspace, []toolchainv1alpha1.SpaceBinding{})
			assert.EqualError(t, err, "error listing spacebindings")
			assert.Len(t, spaceBindings, 0, "invalid number of spacebindings")
		})

		t.Run("error getting parent space", func(t *testing.T) {
			// given
			mur := NewMasterUserRecord(t, "johnny", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
			myspace := spacetest.NewSpace(test.HostOperatorNs, "myspace", spacetest.WithSpecParentSpace("myparentspace"))
			listSpaceBindingsFunc := func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(mur, myspace, "johnny", spacebinding.WithRole("admin")),
				}, nil
			}
			getSpaceFunc := func(spaceName string) (*toolchainv1alpha1.Space, error) {
				return nil, fmt.Errorf("mock error")
			}

			// when
			spaceBindingLister := spacebinding.NewLister(listSpaceBindingsFunc, getSpaceFunc)

			// then
			spaceBindings, err := spaceBindingLister.ListForSpace(myspace, []toolchainv1alpha1.SpaceBinding{})
			assert.EqualError(t, err, "unable to get parent-space: mock error")
			assert.Len(t, spaceBindings, 1, "invalid number of spacebindings") // expect 1 spacebinding to be returned
		})

	})
}
