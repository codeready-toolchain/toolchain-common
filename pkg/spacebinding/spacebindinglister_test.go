package spacebinding_test

import (
	"errors"
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

// Check pkg/spacebinding/spacebindinglister_test.md for more details about this test
func TestNewSpaceBindingLister(t *testing.T) {

	t.Run("recursive list for space", func(t *testing.T) {
		// given
		catwomenMur := NewMasterUserRecord(t, "catwomen", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
		batmanMur := NewMasterUserRecord(t, "batman", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
		robinMur := NewMasterUserRecord(t, "robin", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
		// we have a parent space
		spaceA := spacetest.NewSpace(test.HostOperatorNs, "Space-A",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithLabel(toolchainv1alpha1.SpaceCreatorLabelKey, "catwomen"),
		)
		spaceB := spacetest.NewSpace(test.HostOperatorNs, "Space-B",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(spaceA.GetName()),
		)
		spaceD := spacetest.NewSpace(test.HostOperatorNs, "Space-D",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(spaceB.GetName()),
		)
		spaceE := spacetest.NewSpace(test.HostOperatorNs, "Space-E",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(spaceD.GetName()),
		)

		spaceC := spacetest.NewSpace(test.HostOperatorNs, "Space-C",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(spaceB.GetName()),
		)
		spaceF := spacetest.NewSpace(test.HostOperatorNs, "Space-F",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(spaceC.GetName()),
		)
		errorspace := spacetest.NewSpace(test.HostOperatorNs, "errorspace",
			spacetest.WithSpecParentSpace("invalidparent"))
		errorspacebindings := spacetest.NewSpace(test.HostOperatorNs, "errorspacebindings")
		spaces := map[string]*toolchainv1alpha1.Space{
			spaceA.GetName(): spaceA,
			spaceB.GetName(): spaceB,
			spaceC.GetName(): spaceC,
			spaceD.GetName(): spaceD,
			spaceE.GetName(): spaceE,
			spaceF.GetName(): spaceF,
		}
		// the getSpaceFunc returns the space based on the given name
		getSpaceFunc := func(spaceName string) (*toolchainv1alpha1.Space, error) {
			if space, found := spaces[spaceName]; found {
				return space, nil
			}
			if spaceName == "invalidparent" {
				return nil, errors.New("mock error")
			}
			return nil, fmt.Errorf("space not found: %s", spaceName)
		}

		// listParentSpaceBindingFunc returns the specific spacebindings for the given space
		listParentSpaceBindingFunc := func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
			switch spaceName {
			case spaceA.GetName():
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(catwomenMur, spaceA, spaceA.Name, spacebinding.WithRole("admin")),
				}, nil
			case spaceB.GetName():
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(batmanMur, spaceB, spaceB.Name, spacebinding.WithRole("maintainer")),
				}, nil
			case spaceC.GetName():
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(batmanMur, spaceC, spaceC.Name, spacebinding.WithRole("maintainer")),
				}, nil
			case spaceD.GetName():
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(batmanMur, spaceD, spaceD.Name, spacebinding.WithRole("admin")),
				}, nil
			case spaceE.GetName():
				return []toolchainv1alpha1.SpaceBinding{}, nil
			case spaceF.GetName():
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(robinMur, spaceF, spaceF.Name, spacebinding.WithRole("viewer")),
				}, nil
			case errorspacebindings.GetName():
				// test error case
				return nil, errors.New("error listing spacebindings")
			default:
				return nil, nil
			}
		}

		tests := map[string]struct {
			space                 *toolchainv1alpha1.Space
			getSpaceFunc          func(spaceName string) (*toolchainv1alpha1.Space, error)
			listSpaceBindingsFunc func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error)
			expectedSpaceBindings []toolchainv1alpha1.SpaceBinding
			expectedErr           string
		}{
			"for Space-A": {
				space:                 spaceA,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect to have one spacebinding for the root space
					*spacebinding.NewSpaceBinding(catwomenMur, spaceA, spaceA.Name, spacebinding.WithRole("admin")),
				},
			},
			"for Space-B": {
				space:                 spaceB,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect spacebinding inherited from space-A
					*spacebinding.NewSpaceBinding(catwomenMur, spaceA, spaceA.Name, spacebinding.WithRole("admin")),
					// and specific one from space-B
					*spacebinding.NewSpaceBinding(batmanMur, spaceB, spaceB.Name, spacebinding.WithRole("maintainer")),
				},
			},
			"for Space-C": {
				space:                 spaceC,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect spacebinding inherited from space-A
					*spacebinding.NewSpaceBinding(catwomenMur, spaceA, spaceA.Name, spacebinding.WithRole("admin")),
					// and specific one from space-C (which overrides the one from space-B even if it has the same role)
					*spacebinding.NewSpaceBinding(batmanMur, spaceC, spaceC.Name, spacebinding.WithRole("maintainer")),
				},
			},
			"for Space-D": {
				space:                 spaceD,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect spacebinding inherited from space-A
					*spacebinding.NewSpaceBinding(catwomenMur, spaceA, spaceA.Name, spacebinding.WithRole("admin")),
					// and the one from space-D which overrides the one from space-B
					*spacebinding.NewSpaceBinding(batmanMur, spaceD, spaceD.Name, spacebinding.WithRole("admin")),
				},
			},
			"for Space-E": {
				space:                 spaceD,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// both are inherited
					*spacebinding.NewSpaceBinding(catwomenMur, spaceA, spaceA.Name, spacebinding.WithRole("admin")),
					*spacebinding.NewSpaceBinding(batmanMur, spaceD, spaceD.Name, spacebinding.WithRole("admin")),
				},
			},
			"for Space-F": {
				space:                 spaceF,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// inherited from space-A
					*spacebinding.NewSpaceBinding(catwomenMur, spaceA, spaceA.Name, spacebinding.WithRole("admin")),
					// inherited from space-C
					*spacebinding.NewSpaceBinding(batmanMur, spaceC, spaceC.Name, spacebinding.WithRole("maintainer")),
					// specific one only for this space
					*spacebinding.NewSpaceBinding(robinMur, spaceF, spaceF.Name, spacebinding.WithRole("viewer")),
				},
			},
			"error listing spacebindings": {
				space:                 errorspacebindings,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{}, // empty
				expectedErr:           "error listing spacebindings",
			},
			"error while getting parent-space": {
				space:                 errorspace,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listParentSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{},
				expectedErr:           "unable to get parent-space: mock error",
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
				} else {
					assert.NoError(t, err)
					assert.Len(t, spaceBindings, len(tc.expectedSpaceBindings), "invalid number of spacebindings")
					for _, expectedSpaceBinding := range tc.expectedSpaceBindings {
						found := false
						for _, actualSpaceBinding := range spaceBindings {
							if actualSpaceBinding.GetName() != expectedSpaceBinding.GetName() {
								continue
							}
							found = true
							assert.Equal(t, expectedSpaceBinding.Spec.MasterUserRecord, actualSpaceBinding.Spec.MasterUserRecord)
							assert.Equal(t, expectedSpaceBinding.Spec.Space, actualSpaceBinding.Spec.Space)
							assert.Equal(t, expectedSpaceBinding.Spec.SpaceRole, actualSpaceBinding.Spec.SpaceRole)

							require.NotNil(t, actualSpaceBinding.Labels)
							assert.Equal(t, expectedSpaceBinding.Labels[toolchainv1alpha1.SpaceCreatorLabelKey], actualSpaceBinding.Labels[toolchainv1alpha1.SpaceCreatorLabelKey])
							assert.Equal(t, expectedSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingMasterUserRecordLabelKey], actualSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingMasterUserRecordLabelKey])
							assert.Equal(t, expectedSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingSpaceLabelKey], actualSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingSpaceLabelKey])

						}
						if !found {
							t.Logf("expected spacebinding %s not found.", expectedSpaceBinding.GetName())
							t.FailNow()
						}
					}
				}
			})
		}
	})

	t.Run("recursive list for space with disable Inheritance", func(t *testing.T) {
		// given
		spaceAMur := NewMasterUserRecord(t, "space-a-user", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
		spaceBMur := NewMasterUserRecord(t, "space-b-user", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
		spaceCMur := NewMasterUserRecord(t, "space-c-user", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
		spaceDMur := NewMasterUserRecord(t, "space-d-user", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
		spaceFMur := NewMasterUserRecord(t, "space-f-user", TargetCluster(test.MemberClusterName), TierName("deactivate90"))
		// we have a parent space
		spaceA := spacetest.NewSpace(test.HostOperatorNs, "Space-A",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithDisableInheritance(false),
			spacetest.WithLabel(toolchainv1alpha1.SpaceCreatorLabelKey, "space-a-user"),
		)
		spaceB := spacetest.NewSpace(test.HostOperatorNs, "Space-B",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(spaceA.GetName()),
			spacetest.WithDisableInheritance(true),
		)
		spaceC := spacetest.NewSpace(test.HostOperatorNs, "Space-C",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(spaceB.GetName()),
			spacetest.WithDisableInheritance(false),
		)
		spaceD := spacetest.NewSpace(test.HostOperatorNs, "Space-D",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(spaceC.GetName()),
			spacetest.WithDisableInheritance(false),
		)
		spaceE := spacetest.NewSpace(test.HostOperatorNs, "Space-E",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(spaceD.GetName()),
			spacetest.WithDisableInheritance(true),
		)
		spaceF := spacetest.NewSpace(test.HostOperatorNs, "Space-F",
			spacetest.WithTierName("advanced"),
			spacetest.WithSpecTargetCluster(test.MemberClusterName),
			spacetest.WithSpecParentSpace(spaceE.GetName()),
			spacetest.WithDisableInheritance(false),
		)
		spaces := map[string]*toolchainv1alpha1.Space{
			spaceA.GetName(): spaceA,
			spaceB.GetName(): spaceB,
			spaceC.GetName(): spaceC,
			spaceD.GetName(): spaceD,
			spaceE.GetName(): spaceE,
			spaceF.GetName(): spaceF,
		}
		// the getSpaceFunc returns the space based on the given name
		getSpaceFunc := func(spaceName string) (*toolchainv1alpha1.Space, error) {
			if space, found := spaces[spaceName]; found {
				return space, nil
			}
			return nil, fmt.Errorf("space not found: %s", spaceName)
		}

		// listSpaceBindingFunc returns the specific spacebindings for the given space
		listSpaceBindingFunc := func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error) {
			switch spaceName {
			case spaceA.GetName():
				fmt.Println("return binding for SpaceA")
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(spaceAMur, spaceA, spaceA.Name, spacebinding.WithRole("admin")),
				}, nil
			case spaceB.GetName():
				fmt.Println("return binding for SpaceB")
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(spaceBMur, spaceB, spaceB.Name, spacebinding.WithRole("viewer")),
				}, nil
			case spaceC.GetName():
				fmt.Println("return binding for SpaceC")
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(spaceCMur, spaceC, spaceC.Name, spacebinding.WithRole("maintainer")),
				}, nil
			case spaceD.GetName():
				fmt.Println("return binding for SpaceD")
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(spaceDMur, spaceD, spaceD.Name, spacebinding.WithRole("admin")),
				}, nil
			case spaceE.GetName():
				fmt.Println("return no binding for SpaceE")
				return []toolchainv1alpha1.SpaceBinding{}, nil
			case spaceF.GetName():
				fmt.Println("return binding for SpaceF")
				return []toolchainv1alpha1.SpaceBinding{
					*spacebinding.NewSpaceBinding(spaceFMur, spaceF, spaceF.Name, spacebinding.WithRole("viewer")),
				}, nil
			default:
				return nil, nil
			}
		}

		tests := map[string]struct {
			space                 *toolchainv1alpha1.Space
			getSpaceFunc          func(spaceName string) (*toolchainv1alpha1.Space, error)
			listSpaceBindingsFunc func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error)
			expectedSpaceBindings []toolchainv1alpha1.SpaceBinding
			expectedErr           string
		}{
			"for Space-A": {
				space:                 spaceA,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect to have one spacebinding for the root space
					*spacebinding.NewSpaceBinding(spaceAMur, spaceA, spaceA.Name, spacebinding.WithRole("admin")),
				},
			},
			"for Space-B": {
				space:                 spaceB,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// only expect space binding form space-B
					*spacebinding.NewSpaceBinding(spaceBMur, spaceB, spaceB.Name, spacebinding.WithRole("viewer")),
				},
			},
			"for Space-C": {
				space:                 spaceC,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect spacebinding inherited from space-B
					*spacebinding.NewSpaceBinding(spaceBMur, spaceB, spaceB.Name, spacebinding.WithRole("viewer")),
					// and specific one from space-C (which overrides the one from space-B even if it has the same role)
					*spacebinding.NewSpaceBinding(spaceCMur, spaceC, spaceC.Name, spacebinding.WithRole("maintainer")),
				},
			},
			"for Space-D": {
				space:                 spaceD,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// we expect spacebinding inherited from space-B
					*spacebinding.NewSpaceBinding(spaceBMur, spaceB, spaceB.Name, spacebinding.WithRole("viewer")),
					// we expect spacebinding inherited from space-C
					*spacebinding.NewSpaceBinding(spaceCMur, spaceC, spaceC.Name, spacebinding.WithRole("maintainer")),
					// and the one from space-D
					*spacebinding.NewSpaceBinding(spaceDMur, spaceD, spaceD.Name, spacebinding.WithRole("admin")),
				},
			},
			"for Space-E": {
				space:                 spaceE,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{},
			},
			"for Space-F": {
				space:                 spaceF,
				getSpaceFunc:          getSpaceFunc,
				listSpaceBindingsFunc: listSpaceBindingFunc,
				expectedSpaceBindings: []toolchainv1alpha1.SpaceBinding{
					// specific one only for this space
					*spacebinding.NewSpaceBinding(spaceFMur, spaceF, spaceF.Name, spacebinding.WithRole("viewer")),
				},
			},
		}
		fmt.Println("\nStart DI test")
		for k, tc := range tests {
			t.Run(k, func(t *testing.T) {

				fmt.Println(fmt.Sprintf("\nCheck space %s", tc.space.GetName()))
				// when
				spaceBindingLister := spacebinding.NewLister(tc.listSpaceBindingsFunc, tc.getSpaceFunc)

				// then
				spaceBindings, err := spaceBindingLister.ListForSpace(tc.space, []toolchainv1alpha1.SpaceBinding{})
				if tc.expectedErr != "" {
					assert.EqualError(t, err, tc.expectedErr)
				} else {
					assert.NoError(t, err)
					assert.Len(t, spaceBindings, len(tc.expectedSpaceBindings), "invalid number of spacebindings for %s", tc.space.GetName())
					for _, expectedSpaceBinding := range tc.expectedSpaceBindings {
						found := false
						for _, actualSpaceBinding := range spaceBindings {
							if actualSpaceBinding.GetName() != expectedSpaceBinding.GetName() {
								continue
							}
							found = true
							assert.Equal(t, expectedSpaceBinding.Spec.MasterUserRecord, actualSpaceBinding.Spec.MasterUserRecord)
							assert.Equal(t, expectedSpaceBinding.Spec.Space, actualSpaceBinding.Spec.Space)
							assert.Equal(t, expectedSpaceBinding.Spec.SpaceRole, actualSpaceBinding.Spec.SpaceRole)

							require.NotNil(t, actualSpaceBinding.Labels)
							assert.Equal(t, expectedSpaceBinding.Labels[toolchainv1alpha1.SpaceCreatorLabelKey], actualSpaceBinding.Labels[toolchainv1alpha1.SpaceCreatorLabelKey])
							assert.Equal(t, expectedSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingMasterUserRecordLabelKey], actualSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingMasterUserRecordLabelKey])
							assert.Equal(t, expectedSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingSpaceLabelKey], actualSpaceBinding.Labels[toolchainv1alpha1.SpaceBindingSpaceLabelKey])

						}
						if !found {
							t.Logf("expected spacebinding %s not found.", expectedSpaceBinding.GetName())
							t.FailNow()
						}
					}
				}
			})
		}
	})
}
