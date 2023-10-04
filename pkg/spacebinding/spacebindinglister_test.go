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
		// given
		mur := NewMasterUserRecord(t, "johnny", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
		// we have a parent space
		parentSpace := spacetest.NewSpace(test.HostOperatorNs, "smith",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithLabel(toolchainv1alpha1.SpaceCreatorLabelKey, "johnny"),
		)
		// we have 2 subspaces for the above parent space
		childSpace1 := spacetest.NewSpace(test.HostOperatorNs, "smith-child1",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(parentSpace.GetName()),
		)
		childSpace2 := spacetest.NewSpace(test.HostOperatorNs, "smith-child2",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(parentSpace.GetName()),
		)
		// this is a grandchild
		grandChildSpace1 := spacetest.NewSpace(test.HostOperatorNs, "smith-grandChild1",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(childSpace1.GetName()),
		)
		spaces := map[string]*toolchainv1alpha1.Space{
			parentSpace.Name: parentSpace,
			childSpace1.Name: childSpace1, childSpace2.Name: childSpace2,
			grandChildSpace1.Name: grandChildSpace1,
		}
		// the defaultGetSpaceFunc returns the space based on the given name
		defaultGetSpaceFunc := func(spaceName string) (*toolchainv1alpha1.Space, error) {
			if space, found := spaces[spaceName]; found {
				return space, nil
			}
			return nil, fmt.Errorf("space not found: %s", spaceName)
		}

		// listParentSpaceBindingFunc returns the spacebinding for the parentSpace
		listParentSpaceBindingFunc := func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
			return []toolchainv1alpha1.SpaceBinding{
				*spacebinding.NewSpaceBinding(mur, parentSpace, parentSpace.Name, spacebinding.WithRole("admin")),
			}, nil
		}

		tests := map[string]struct {
			space                 *toolchainv1alpha1.Space
			getSpaceFunc          func(spaceName string) (*toolchainv1alpha1.Space, error)
			listSpaceBindingsFunc func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error)
			expectedSpaceBindings []toolchainv1alpha1.SpaceBinding
			expectedErr           string
		}{
			"parentSpace has it's own spacebinding": {
				space:                 parentSpace,
				getSpaceFunc:          defaultGetSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect to have only one spacebinding for the parent space
					*spacebinding.NewSpaceBinding(mur, parentSpace, parentSpace.Name, spacebinding.WithRole("admin")),
				},
			},
			"spacebinding for childSpace1 is inherited from parentSpace": {
				space:                 childSpace1,
				getSpaceFunc:          defaultGetSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect to have only one spacebinding from the parent space
					*spacebinding.NewSpaceBinding(mur, parentSpace, parentSpace.Name, spacebinding.WithRole("admin")),
				},
			},
			"override parentSpace-binding in childSpace2": {
				space:        childSpace2,
				getSpaceFunc: defaultGetSpaceFunc,
				listSpaceBindingsFunc: func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
					switch spaceName {
					case parentSpace.GetName():
						return []toolchainv1alpha1.SpaceBinding{
							*spacebinding.NewSpaceBinding(mur, parentSpace, parentSpace.Name, spacebinding.WithRole("admin")),
						}, nil
					case childSpace2.GetName():
						return []toolchainv1alpha1.SpaceBinding{
							*spacebinding.NewSpaceBinding(mur, childSpace2, childSpace2.Name, spacebinding.WithRole("viewer")),
						}, nil
					default:
						return nil, nil
					}
				},
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect user to have viewer role on childSpace2
					*spacebinding.NewSpaceBinding(mur, childSpace2, childSpace2.Name, spacebinding.WithRole("viewer")),
				},
			},
			"no spacebinding on child1 and grandChildSpace1, spacebinding for grandChildSpace1 is inherited from parentSpace": {
				space:                 grandChildSpace1,
				getSpaceFunc:          defaultGetSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect to have only one spacebinding from the parent space
					*spacebinding.NewSpaceBinding(mur, parentSpace, parentSpace.Name, spacebinding.WithRole("admin")),
				},
			},
			"no spacebinding on child1 but we do have one on grandChildSpace1 which overrides parentSpace-binding": {
				space:        grandChildSpace1,
				getSpaceFunc: defaultGetSpaceFunc,
				listSpaceBindingsFunc: func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
					switch spaceName {
					case parentSpace.GetName():
						return []toolchainv1alpha1.SpaceBinding{
							*spacebinding.NewSpaceBinding(mur, parentSpace, parentSpace.Name, spacebinding.WithRole("admin")),
						}, nil
					case grandChildSpace1.GetName():
						return []toolchainv1alpha1.SpaceBinding{
							*spacebinding.NewSpaceBinding(mur, grandChildSpace1, grandChildSpace1.Name, spacebinding.WithRole("viewer")),
						}, nil
					default:
						return nil, nil
					}
				},
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect user to have maintainer role on grandChildSpace1
					*spacebinding.NewSpaceBinding(mur, grandChildSpace1, grandChildSpace1.Name, spacebinding.WithRole("viewer")),
				},
			},
			"spacebinding on grandChildSpace1 overrides child1-binding and parentSpace-binding": {
				space:        grandChildSpace1,
				getSpaceFunc: defaultGetSpaceFunc,
				listSpaceBindingsFunc: func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
					// we have specific spacebinding for each level with different roles
					switch spaceName {
					case parentSpace.GetName():
						return []toolchainv1alpha1.SpaceBinding{
							*spacebinding.NewSpaceBinding(mur, grandChildSpace1, grandChildSpace1.Name, spacebinding.WithRole("admin")),
						}, nil
					case childSpace1.GetName():
						return []toolchainv1alpha1.SpaceBinding{
							*spacebinding.NewSpaceBinding(mur, childSpace1, childSpace1.Name, spacebinding.WithRole("maintainer")),
						}, nil
					case grandChildSpace1.GetName():
						return []toolchainv1alpha1.SpaceBinding{
							*spacebinding.NewSpaceBinding(mur, grandChildSpace1, grandChildSpace1.Name, spacebinding.WithRole("viewer")),
						}, nil
					default:
						return nil, nil
					}
				},
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect user to have viewer role on grandChildSpace1
					*spacebinding.NewSpaceBinding(mur, grandChildSpace1, grandChildSpace1.Name, spacebinding.WithRole("viewer")),
				},
			},
			"error listing spacebindings": {
				space:        spacetest.NewSpace(test.HostOperatorNs, "myspace"),
				getSpaceFunc: defaultGetSpaceFunc,
				listSpaceBindingsFunc: func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
					return nil, fmt.Errorf("error listing spacebindings")
				},
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{}, // empty
				expectedErr:           "error listing spacebindings",
			},
			"error getting parentSpace": {
				space: childSpace2,
				getSpaceFunc: func(spaceName string) (*toolchainv1alpha1.Space, error) {
					return nil, fmt.Errorf("mock error")
				},
				listSpaceBindingsFunc: func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
					return []toolchainv1alpha1.SpaceBinding{
						*spacebinding.NewSpaceBinding(mur, childSpace2, childSpace2.GetName(), spacebinding.WithRole("admin")),
					}, nil
				},
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(mur, childSpace2, childSpace2.GetName(), spacebinding.WithRole("admin")),
				},
				expectedErr: "unable to get parent-space: mock error",
			},
		}

		for k, tc := range tests {
			t.Run(k, func(t *testing.T) {
				// when
				spaceBindingLister := spacebinding.NewLister(tc.listSpaceBindingsFunc, tc.getSpaceFunc)

				// then
				spaceBindings, err := spaceBindingLister.ListForSpace(tc.space, []toolchainv1alpha1.SpaceBinding{})
				if tc.expectedErr != "" {
					assert.EqualError(t, err, tc.expectedErr)
					assert.Len(t, spaceBindings, len(tc.expectedSpaceBindings), "invalid number of spacebindings")
				} else {
					assert.NoError(t, err)
					assert.Len(t, spaceBindings, len(tc.expectedSpaceBindings), "invalid number of spacebindings")
					actualSpaceBinding := spaceBindings[0]
					expectedSpaceBinding := tc.expectedSpaceBindings[0]
					assert.Equal(t, mur.Name, actualSpaceBinding.Spec.MasterUserRecord)
					assert.Equal(t, expectedSpaceBinding.Spec.Space, actualSpaceBinding.Spec.Space)
					assert.Equal(t, expectedSpaceBinding.Spec.SpaceRole, actualSpaceBinding.Spec.SpaceRole)

					require.NotNil(t, actualSpaceBinding.Labels)
					assert.Equal(t, expectedSpaceBinding.Labels[toolchainv1alpha1.SpaceCreatorLabelKey], actualSpaceBinding.Labels[toolchainv1alpha1.SpaceCreatorLabelKey])
					assert.Equal(t, expectedSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingMasterUserRecordLabelKey], actualSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingMasterUserRecordLabelKey])
					assert.Equal(t, expectedSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingSpaceLabelKey], actualSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingSpaceLabelKey])
				}
			})
		}
	})
}
