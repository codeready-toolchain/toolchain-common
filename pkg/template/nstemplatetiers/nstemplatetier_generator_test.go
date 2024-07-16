package nstemplatetiers

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"testing"
	texttemplate "text/template"

	"github.com/google/uuid"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	commonclient "github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ghodss/yaml"
	templatev1 "github.com/openshift/api/template/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

//go:embed testdata/nstemplatetiers*
var testTemplateFiles embed.FS

var expectedTestTiers = map[string]bool{
	"advanced":  true, // tier_name: true/false (if based on the other tier)
	"base":      false,
	"nocluster": false,
	"appstudio": false,
}

func getTestMetadata() map[string]string {
	return map[string]string{
		"advanced/based_on_tier":          "abcd123",
		"base/tier":                       "123456z",
		"base/cluster":                    "654321a",
		"base/ns_dev":                     "123456b",
		"base/ns_stage":                   "123456c",
		"base/spacerole_admin":            "123456d",
		"nocluster/tier":                  "1234567y",
		"nocluster/ns_dev":                "123456j",
		"nocluster/ns_stage":              "1234567", // here, a number string
		"nocluster/spacerole_admin":       "123456k",
		"appstudio/tier":                  "123456z",
		"appstudio/cluster":               "654321a",
		"appstudio/ns_tenant":             "123456b",
		"appstudio/spacerole_admin":       "123456c",
		"appstudio/spacerole_maintainer":  "123456d",
		"appstudio/spacerole_contributor": "123456e",
	}
}

func nsTypes(tier string) []string {
	switch tier {
	case "appstudio":
		return []string{"tenant"}
	case "appstudio-env":
		return []string{"env"}
	case "base1ns", "base1nsnoidling", "base1ns6didler", "test":
		return []string{"dev"}
	default:
		return []string{"dev", "stage"}
	}
}

func roles(tier string) []string {
	switch tier {
	case "appstudio", "appstudio-env":
		return []string{"admin", "maintainer", "contributor"}
	default:
		return []string{"admin"}
	}
}

func isNamespaceType(expectedTiers map[string]bool, typeName string) bool {
	for _, tier := range tiers(expectedTiers) {
		for _, t := range nsTypes(tier) {
			if t == typeName {
				return true
			}
		}
	}
	return false
}

func isSpaceRole(expectedTiers map[string]bool, roleName string) bool {
	for _, tier := range tiers(expectedTiers) {
		for _, r := range roles(tier) {
			if r == roleName {
				return true
			}
		}
	}
	return false
}

func tiers(expectedTiers map[string]bool) []string {
	tt := make([]string, 0, len(expectedTiers))
	for tier := range expectedTiers {
		tt = append(tt, tier)
	}
	return tt
}

func basedOnOtherTier(expectedTiers map[string]bool, tier string) bool {
	return expectedTiers[tier]
}

func getTestTemplates(t *testing.T) map[string][]byte {
	templates := map[string][]byte{}
	err := fs.WalkDir(testTemplateFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		file, err := testTemplateFiles.ReadFile(path)
		if err != nil {
			return err
		}
		templates[filepath.Join(filepath.Base(filepath.Dir(path)), d.Name())] = file
		return nil
	})
	require.NoError(t, err)
	return templates
}

func TestGenerateTiers(t *testing.T) {

	s := addToScheme(t)
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	t.Run("ok", func(t *testing.T) {

		expectedTemplateRefs := map[string]map[string]interface{}{
			"advanced": {
				"clusterresources": "advanced-clusterresources-abcd123-654321a",
				"namespaces": []string{
					"advanced-dev-abcd123-123456b",
					"advanced-stage-abcd123-123456c",
				},
				"spaceRoles": map[string]string{
					"admin": "advanced-admin-abcd123-123456d",
				},
			},
			"base": {
				"clusterresources": "base-clusterresources-654321a-654321a",
				"namespaces": []string{
					"base-dev-123456b-123456b",
					"base-stage-123456c-123456c",
				},
				"spaceRoles": map[string]string{
					"admin": "base-admin-123456d-123456d",
				},
			},
			"nocluster": {
				"namespaces": []string{
					"nocluster-dev-123456j-123456j",
					"nocluster-stage-1234567-1234567",
				},
				"spaceRoles": map[string]string{
					"admin": "nocluster-admin-123456k-123456k",
				},
			},
			"appstudio": {
				"clusterresources": "appstudio-clusterresources-654321a-654321a",
				"namespaces": []string{
					"appstudio-tenant-123456b-123456b",
				},
				"spaceRoles": map[string]string{
					"admin":       "appstudio-admin-123456c-123456c",
					"maintainer":  "appstudio-maintainer-123456d-123456d",
					"contributor": "appstudio-contributor-123456e-123456e",
				},
			},
		}

		t.Run("create only", func(t *testing.T) {
			// given
			namespace := "host-operator" + uuid.NewString()[:7]
			clt := test.NewFakeClient(t)
			// verify that no NSTemplateTier resources exist prior to creation
			nsTmplTiers := toolchainv1alpha1.NSTemplateTierList{}
			err := clt.List(context.TODO(), &nsTmplTiers, runtimeclient.InNamespace(namespace))
			require.NoError(t, err)
			require.Empty(t, nsTmplTiers.Items)
			// verify that no TierTemplate resources exist prior to creation
			tierTmpls := toolchainv1alpha1.TierTemplateList{}
			err = clt.List(context.TODO(), &tierTmpls, runtimeclient.InNamespace(namespace))
			require.NoError(t, err)
			require.Empty(t, tierTmpls.Items)

			// when
			err = GenerateTiers(s, ensureObjectFuncForClient(clt), namespace, getTestMetadata(), getTestTemplates(t))

			// then
			require.NoError(t, err)

			// verify that TierTemplates were created
			tierTmpls = toolchainv1alpha1.TierTemplateList{}
			err = clt.List(context.TODO(), &tierTmpls, runtimeclient.InNamespace(namespace))
			require.NoError(t, err)
			require.Len(t, tierTmpls.Items, 16) // 4 items for advanced and base tiers + 3 for nocluster tier + 5 for appstudio
			names := []string{}
			for _, tierTmpl := range tierTmpls.Items {
				names = append(names, tierTmpl.Name)
			}
			require.ElementsMatch(t, []string{
				"advanced-clusterresources-abcd123-654321a",
				"advanced-dev-abcd123-123456b",
				"advanced-stage-abcd123-123456c",
				"advanced-admin-abcd123-123456d",
				"base-clusterresources-654321a-654321a",
				"base-dev-123456b-123456b",
				"base-stage-123456c-123456c",
				"base-admin-123456d-123456d",
				"nocluster-dev-123456j-123456j",
				"nocluster-stage-1234567-1234567",
				"nocluster-admin-123456k-123456k",
				"appstudio-clusterresources-654321a-654321a",
				"appstudio-tenant-123456b-123456b",
				"appstudio-admin-123456c-123456c",
				"appstudio-maintainer-123456d-123456d",
				"appstudio-contributor-123456e-123456e",
			}, names)

			// verify that 4 NSTemplateTier CRs were created:
			for _, tierName := range []string{"advanced", "base", "nocluster", "appstudio"} {
				tier := toolchainv1alpha1.NSTemplateTier{}
				err = clt.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: tierName}, &tier)
				require.NoError(t, err)
				assert.Equal(t, int64(1), tier.ObjectMeta.Generation)

				// check the `clusterresources` templateRef
				if tier.Name == "nocluster" {
					assert.Nil(t, tier.Spec.ClusterResources) // "nocluster" tier should not have cluster resources set
				} else {
					require.NotNil(t, tier.Spec.ClusterResources)
					assert.Equal(t, expectedTemplateRefs[tierName]["clusterresources"], tier.Spec.ClusterResources.TemplateRef)
				}

				// check the `namespaces` templateRefs
				actualNamespaceTmplRefs := make([]string, len(tier.Spec.Namespaces))
				for i, ns := range tier.Spec.Namespaces {
					actualNamespaceTmplRefs[i] = ns.TemplateRef
				}
				assert.ElementsMatch(t, expectedTemplateRefs[tierName]["namespaces"], actualNamespaceTmplRefs)

				// check the `spaceRoles` templateRefs
				actualSpaceRoleTmplRefs := make(map[string]string, len(tier.Spec.SpaceRoles))
				for i, r := range tier.Spec.SpaceRoles {
					actualSpaceRoleTmplRefs[i] = r.TemplateRef
				}
				for role, tmpl := range expectedTemplateRefs[tierName]["spaceRoles"].(map[string]string) {
					assert.Equal(t, tmpl, actualSpaceRoleTmplRefs[role])
				}
			}
		})

		t.Run("create then update with same tier templates", func(t *testing.T) {
			// given
			namespace := "host-operator" + uuid.NewString()[:7]
			clt := test.NewFakeClient(t)

			// when
			err := GenerateTiers(s, ensureObjectFuncForClient(clt), namespace, getTestMetadata(), getTestTemplates(t))
			require.NoError(t, err)

			// when calling CreateOrUpdateResources a second time
			err = GenerateTiers(s, ensureObjectFuncForClient(clt), namespace, getTestMetadata(), getTestTemplates(t))

			// then
			require.NoError(t, err)
			// verify that all TierTemplate CRs were updated
			tierTmpls := toolchainv1alpha1.TierTemplateList{}
			err = clt.List(context.TODO(), &tierTmpls, runtimeclient.InNamespace(namespace))
			require.NoError(t, err)
			require.Len(t, tierTmpls.Items, 16) // 4 items for advanced and base tiers + 3 for nocluster tier + 4 for appstudio
			for _, tierTmpl := range tierTmpls.Items {
				assert.Equal(t, int64(1), tierTmpl.ObjectMeta.Generation) // unchanged
			}

			// verify that 4 NSTemplateTier CRs were created:
			for _, tierName := range []string{"advanced", "base", "nocluster", "appstudio"} {
				tier := toolchainv1alpha1.NSTemplateTier{}
				err = clt.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: tierName}, &tier)
				require.NoError(t, err)
				assert.Equal(t, int64(1), tier.ObjectMeta.Generation)

				// check the `clusterresources` templateRef
				if tier.Name == "nocluster" {
					assert.Nil(t, tier.Spec.ClusterResources)
				} else {
					require.NotNil(t, tier.Spec.ClusterResources)
					assert.Equal(t, expectedTemplateRefs[tierName]["clusterresources"], tier.Spec.ClusterResources.TemplateRef)
				}

				// check the `namespaces` templateRefs
				actualTemplateRefs := make([]string, len(tier.Spec.Namespaces))
				for i, ns := range tier.Spec.Namespaces {
					actualTemplateRefs[i] = ns.TemplateRef
				}
				assert.ElementsMatch(t, expectedTemplateRefs[tierName]["namespaces"], actualTemplateRefs)

				// check the `spaceRoles` templateRefs
				actualSpaceRoleTmplRefs := make(map[string]string, len(tier.Spec.SpaceRoles))
				for i, r := range tier.Spec.SpaceRoles {
					actualSpaceRoleTmplRefs[i] = r.TemplateRef
				}
				for role, tmpl := range expectedTemplateRefs[tierName]["spaceRoles"].(map[string]string) {
					assert.Equal(t, tmpl, actualSpaceRoleTmplRefs[role])
				}
			}
		})

		t.Run("create then update with new tier templates", func(t *testing.T) {
			// given
			namespace := "host-operator" + uuid.NewString()[:7]
			clt := test.NewFakeClient(t)

			// when
			err := GenerateTiers(s, ensureObjectFuncForClient(clt), namespace, getTestMetadata(), getTestTemplates(t))
			require.NoError(t, err)

			// given a new set of tier templates (same content but new revisions, which is what we'll want to check here)
			metadata := map[string]string{
				"advanced/based_on_tier":          "111111a",
				"base/cluster":                    "222222a",
				"base/ns_dev":                     "222222b",
				"base/ns_stage":                   "222222c",
				"base/spacerole_admin":            "222222d",
				"nocluster/ns_dev":                "333333a",
				"nocluster/ns_stage":              "333333b",
				"nocluster/spacerole_admin":       "333333c",
				"appstudio/cluster":               "444444a",
				"appstudio/ns_tenant":             "444444b",
				"appstudio/spacerole_admin":       "444444c",
				"appstudio/spacerole_maintainer":  "444444d",
				"appstudio/spacerole_contributor": "444444e",
			}

			// when calling CreateOrUpdateResources a second time
			err = GenerateTiers(s, ensureObjectFuncForClient(clt), namespace, metadata, getTestTemplates(t))

			// then
			require.NoError(t, err)
			// verify that all TierTemplate CRs for the new revisions were created
			tierTmpls := toolchainv1alpha1.TierTemplateList{}
			err = clt.List(context.TODO(), &tierTmpls, runtimeclient.InNamespace(namespace))
			require.NoError(t, err)
			require.Len(t, tierTmpls.Items, 32) // two versions of: 4 items for advanced and base tiers + 3 for nocluster tier + 4 for appstudio
			for _, tierTmpl := range tierTmpls.Items {
				assert.Equal(t, int64(1), tierTmpl.ObjectMeta.Generation) // unchanged
			}

			expectedTemplateRefs := map[string]map[string]interface{}{
				"advanced": {
					"clusterresources": "advanced-clusterresources-111111a-222222a",
					"namespaces": []string{
						"advanced-dev-111111a-222222b",
						"advanced-stage-111111a-222222c",
					},
					"spaceRoles": map[string]string{
						"admin": "advanced-admin-111111a-222222d",
					},
				},
				"base": {
					"clusterresources": "base-clusterresources-222222a-222222a",
					"namespaces": []string{
						"base-dev-222222b-222222b",
						"base-stage-222222c-222222c",
					},
					"spaceRoles": map[string]string{
						"admin": "base-admin-222222d-222222d",
					},
				},
				"nocluster": {
					"namespaces": []string{
						"nocluster-dev-333333a-333333a",
						"nocluster-stage-333333b-333333b",
					},
					"spaceRoles": map[string]string{
						"admin": "nocluster-admin-333333c-333333c",
					},
				},
				"appstudio": {
					"clusterresources": "appstudio-clusterresources-444444a-444444a",
					"namespaces": []string{
						"appstudio-dev-444444a-444444a",
						"appstudio-stage-444444b-444444b",
					},
					"spaceRoles": map[string]string{
						"admin":       "appstudio-admin-444444c-444444c",
						"maintainer":  "appstudio-maintainer-444444d-444444d",
						"contributor": "appstudio-contributor-444444e-444444e",
					},
				},
			}
			// verify that the 3 NStemplateTier CRs were updated
			for _, tierName := range []string{"advanced", "base", "nocluster"} {
				tier := toolchainv1alpha1.NSTemplateTier{}
				err = clt.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: tierName}, &tier)
				require.NoError(t, err)
				assert.Equal(t, int64(2), tier.ObjectMeta.Generation)

				// check the `clusteresources` templateRefs
				if tier.Name == "nocluster" {
					assert.Nil(t, tier.Spec.ClusterResources)
				} else {
					require.NotNil(t, tier.Spec.ClusterResources)
					assert.Equal(t, expectedTemplateRefs[tierName]["clusterresources"], tier.Spec.ClusterResources.TemplateRef)
				}

				// check the `namespaces` templateRefs
				actualTemplateRefs := make([]string, len(tier.Spec.Namespaces))
				for i, ns := range tier.Spec.Namespaces {
					actualTemplateRefs[i] = ns.TemplateRef
				}
				assert.ElementsMatch(t, expectedTemplateRefs[tierName]["namespaces"], actualTemplateRefs)

				// check the `spaceRoles` templateRefs
				actualSpaceRoleTmplRefs := make(map[string]string, len(tier.Spec.SpaceRoles))
				for i, r := range tier.Spec.SpaceRoles {
					actualSpaceRoleTmplRefs[i] = r.TemplateRef
				}
				for role, tmpl := range expectedTemplateRefs[tierName]["spaceRoles"].(map[string]string) {
					assert.Equal(t, tmpl, actualSpaceRoleTmplRefs[role])
				}
			}
		})
	})

	t.Run("failures", func(t *testing.T) {

		namespace := "host-operator" + uuid.NewString()[:7]

		t.Run("nstemplatetiers", func(t *testing.T) {

			t.Run("failed to create nstemplatetiers", func(t *testing.T) {
				// given
				clt := test.NewFakeClient(t)
				clt.MockCreate = func(ctx context.Context, obj runtimeclient.Object, opts ...runtimeclient.CreateOption) error {
					if obj.GetObjectKind().GroupVersionKind().Kind == "NSTemplateTier" {
						// simulate a client/server error
						return errors.Errorf("an error")
					}
					return clt.Client.Create(ctx, obj, opts...)
				}

				// when
				err := GenerateTiers(s, ensureObjectFuncForClient(clt), namespace, getTestMetadata(), getTestTemplates(t))
				// then
				require.Error(t, err)
				assert.Regexp(t, "unable to create or update the '\\w+' NSTemplateTier: unable to create resource of kind: NSTemplateTier, version: v1alpha1: an error", err.Error())
			})

			t.Run("missing tier.yaml file", func(t *testing.T) {
				// given
				clt := test.NewFakeClient(t)
				testTemplates := getTestTemplates(t)
				delete(testTemplates, "appstudio/tier.yaml")

				// when
				err := GenerateTiers(s, ensureObjectFuncForClient(clt), namespace, getTestMetadata(), testTemplates)
				// then
				require.EqualError(t, err, "unable to init NSTemplateTier generator: tier appstudio is missing a tier.yaml file")
			})

			t.Run("failed to update nstemplatetiers", func(t *testing.T) {
				// given
				// initialize the client with an existing `advanced` NSTemplatetier
				clt := test.NewFakeClient(t, &toolchainv1alpha1.NSTemplateTier{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      "advanced",
					},
				})
				clt.MockUpdate = func(ctx context.Context, obj runtimeclient.Object, opts ...runtimeclient.UpdateOption) error {
					if obj.GetObjectKind().GroupVersionKind().Kind == "NSTemplateTier" {
						// simulate a client/server error
						return errors.Errorf("an error")
					}
					return clt.Client.Update(ctx, obj, opts...)
				}

				// when
				err := GenerateTiers(s, ensureObjectFuncForClient(clt), namespace, getTestMetadata(), getTestTemplates(t))

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to create NSTemplateTiers: unable to create or update the 'advanced' NSTemplateTier: unable to create resource of kind: NSTemplateTier, version: v1alpha1: unable to update the resource")
			})

			t.Run("failed to create nstemplatetiers", func(t *testing.T) {
				// given
				clt := test.NewFakeClient(t)
				clt.MockCreate = func(ctx context.Context, obj runtimeclient.Object, opts ...runtimeclient.CreateOption) error {
					if _, ok := obj.(*toolchainv1alpha1.TierTemplate); ok {
						// simulate a client/server error
						return errors.Errorf("an error")
					}
					return clt.Client.Create(ctx, obj, opts...)
				}

				// when
				err := GenerateTiers(s, ensureObjectFuncForClient(clt), namespace, getTestMetadata(), getTestTemplates(t))

				// then
				require.Error(t, err)
				assert.Regexp(t, fmt.Sprintf("unable to create the '\\w+-\\w+-\\w+-\\w+' TierTemplate in namespace '%s'", namespace), err.Error()) // we can't tell for sure which namespace will fail first, but the error should match the given regex
			})
		})
	})
}

func TestLoadTemplatesByTiers(t *testing.T) {

	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	t.Run("ok", func(t *testing.T) {
		t.Run("with test assets", func(t *testing.T) {
			// when
			tmpls, err := loadTemplatesByTiers(getTestMetadata(), getTestTemplates(t))
			// then
			require.NoError(t, err)
			require.Len(t, tmpls, 4)
			require.NotContains(t, "foo", tmpls) // make sure that the `foo: bar` entry was ignored

			for _, tier := range tiers(expectedTestTiers) {
				t.Run(tier, func(t *testing.T) {
					for _, typeName := range nsTypes(tier) {
						t.Run("ns-"+typeName, func(t *testing.T) {
							require.NotNil(t, tmpls[tier])
							if basedOnOtherTier(expectedTestTiers, tier) {
								assert.Empty(t, tmpls[tier].rawTemplates.namespaceTemplates[typeName].revision)
								assert.Empty(t, tmpls[tier].rawTemplates.namespaceTemplates[typeName].content)
							} else {
								assert.NotEmpty(t, tmpls[tier].rawTemplates.namespaceTemplates[typeName].revision)
								assert.NotEmpty(t, tmpls[tier].rawTemplates.namespaceTemplates[typeName].content)
							}
						})
					}
					for _, role := range roles(tier) {
						t.Run("spacerole-"+role, func(t *testing.T) {
							if basedOnOtherTier(expectedTestTiers, tier) {
								assert.Empty(t, tmpls[tier].rawTemplates.spaceroleTemplates[role].revision)
								assert.Empty(t, tmpls[tier].rawTemplates.spaceroleTemplates[role].content)
							} else {
								assert.NotEmpty(t, tmpls[tier].rawTemplates.spaceroleTemplates[role].revision)
								assert.NotEmpty(t, tmpls[tier].rawTemplates.spaceroleTemplates[role].content)
							}
						})
					}
					t.Run("cluster", func(t *testing.T) {
						require.NotNil(t, tmpls[tier].rawTemplates)
						if basedOnOtherTier(expectedTestTiers, tier) {
							assert.Nil(t, tmpls[tier].rawTemplates.clusterTemplate)
						} else if tier != "nocluster" {
							require.NotNil(t, tmpls[tier].rawTemplates.clusterTemplate)
							assert.NotEmpty(t, tmpls[tier].rawTemplates.clusterTemplate.revision)
							assert.NotEmpty(t, tmpls[tier].rawTemplates.clusterTemplate.content)
						} else {
							require.Nil(t, tmpls[tier].rawTemplates.clusterTemplate)
						}
					})
					t.Run("based_on_tier", func(t *testing.T) {
						if basedOnOtherTier(expectedTestTiers, tier) {
							require.NotNil(t, tmpls[tier].basedOnTier)
						} else {
							require.Nil(t, tmpls[tier].basedOnTier)
						}
					})
				})
			}
		})
	})

	t.Run("failures", func(t *testing.T) {

		t.Run("unparseable content", func(t *testing.T) {
			// given
			testTemplates := getTestTemplates(t)
			testTemplates["advanced/based_on_tier.yaml"] = []byte("foo::bar")

			// when
			_, err := loadTemplatesByTiers(getTestMetadata(), testTemplates)
			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unable to unmarshal 'advanced/based_on_tier.yaml': yaml: unmarshal errors:")
		})

		t.Run("invalid name format", func(t *testing.T) {
			// given
			dummyMetadata := map[string]string{
				`.DS_Store`: `metadata.yaml`, // '/advanced/foo.yaml' is not a valid filename
			}
			dummyTemplates := map[string][]byte{
				".DS_Store": []byte(`foo:bar`), // just make sure the asset exists
			}

			// when
			_, err := loadTemplatesByTiers(dummyMetadata, dummyTemplates)
			// then
			require.Error(t, err)
			require.EqualError(t, err, "unable to load templates: invalid name format for file '.DS_Store'")
		})

		t.Run("invalid filename scope", func(t *testing.T) {
			// given
			dummyMetadata := map[string]string{
				`metadata.yaml`: `advanced/foo.yaml`, // '/advanced/foo.yaml' is not a valid filename
			}
			dummyTemplates := map[string][]byte{
				"advanced/foo.yaml": []byte(`foo:bar`), // just make sure the asset exists
			}

			// when
			_, err := loadTemplatesByTiers(dummyMetadata, dummyTemplates)
			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unable to load templates: unknown scope for file 'advanced/foo.yaml'")
		})

		t.Run("should fail when tier contains a mix of based_on_tier.yaml file together with a regular template file", func(t *testing.T) {
			// given
			s := addToScheme(t)
			clt := test.NewFakeClient(t)

			for _, tmplName := range []string{"cluster.yaml", "ns_dev.yaml", "ns_stage.yaml", "tier.yaml"} {
				t.Run("for template name "+tmplName, func(t *testing.T) {
					// given
					filePath := fmt.Sprintf("advanced/%s", tmplName)
					dummyMetadata := getTestMetadata()
					dummyMetadata[filePath] = "123"

					dummyTemplates := getTestTemplates(t)
					dummyTemplates[filePath] = []byte("")

					// when
					_, err := newNSTemplateTierGenerator(s, ensureObjectFuncForClient(clt), test.HostOperatorNs, dummyMetadata, dummyTemplates)

					// then
					require.EqualError(t, err, "the tier advanced contains a mix of based_on_tier.yaml file together with a regular template file")
				})
			}
		})
	})
}

func TestNewNSTemplateTier(t *testing.T) {

	s := scheme.Scheme
	err := toolchainv1alpha1.AddToScheme(s)
	require.NoError(t, err)

	t.Run("ok", func(t *testing.T) {

		t.Run("with test assets", func(t *testing.T) {
			// given
			namespace := "host-operator-" + uuid.NewString()[:7]
			tc, err := newNSTemplateTierGenerator(s, nil, namespace, getTestMetadata(), getTestTemplates(t))
			require.NoError(t, err)
			clusterResourcesRevisions := map[string]string{
				"advanced":  "abcd123-654321a",
				"base":      "654321a-654321a",
				"appstudio": "654321a-654321a",
			}
			namespaceRevisions := map[string]map[string]string{
				"advanced": {
					"dev":   "abcd123-123456b",
					"stage": "abcd123-123456c",
				},
				"base": {
					"dev":   "123456b-123456b",
					"stage": "123456c-123456c",
				},
				"nocluster": {
					"dev":   "123456j-123456j",
					"stage": "1234567-1234567",
				},
				"appstudio": {
					"tenant": "123456b-123456b",
				},
			}
			spaceRoleRevisions := map[string]map[string]string{
				"advanced": {
					"admin": "abcd123-123456d",
				},
				"base": {
					"admin": "123456d-123456d",
				},
				"nocluster": {
					"admin": "123456k-123456k",
				},
				"appstudio": {
					"admin":       "123456c-123456c",
					"maintainer":  "123456d-123456d",
					"contributor": "123456e-123456e",
				},
			}
			for tier := range namespaceRevisions {
				t.Run(tier, func(t *testing.T) {
					// given
					objects := tc.templatesByTier[tier].objects
					require.Len(t, objects, 1, "expected only 1 NSTemplateTier toolchain object")
					// when
					actual := runtimeObjectToNSTemplateTier(t, s, objects[0])

					// then
					expected, err := newNSTemplateTierFromYAML(s, tier, namespace, clusterResourcesRevisions[tier], namespaceRevisions[tier], spaceRoleRevisions[tier])
					require.NoError(t, err)
					// here we don't compare objects because the generated NSTemplateTier
					// has no specific values for the `TypeMeta`: the `APIVersion: toolchain.dev.openshift.com/v1alpha1`
					// and `Kind: NSTemplateTier` should be set by the client using the registered GVK
					assert.Equal(t, expected.ObjectMeta, actual.ObjectMeta)
					assert.Equal(t, expected.Spec, actual.Spec)
				})
			}
		})
	})
}

func TestNewTierTemplate(t *testing.T) {
	s := addToScheme(t)
	namespace := "host-operator-" + uuid.NewString()[:7]

	t.Run("ok", func(t *testing.T) {

		t.Run("with test assets", func(t *testing.T) {
			// given

			// when
			tc, err := newNSTemplateTierGenerator(s, nil, namespace, getTestMetadata(), getTestTemplates(t))

			// then
			require.NoError(t, err)
			decoder := serializer.NewCodecFactory(s).UniversalDeserializer()

			resourceNameRE, err := regexp.Compile(`[a-z0-9\.-]+`)
			require.NoError(t, err)
			for tier, tmpls := range tc.templatesByTier {
				t.Run(tier, func(t *testing.T) {
					for _, actual := range tmpls.tierTemplates {
						assert.Equal(t, namespace, actual.Namespace)
						assert.True(t, resourceNameRE.MatchString(actual.Name)) // verifies that the TierTemplate name complies with the DNS-1123 spec
						assert.NotEmpty(t, actual.Spec.Revision)
						assert.NotEmpty(t, actual.Spec.TierName)
						assert.NotEmpty(t, actual.Spec.Type)
						assert.NotEmpty(t, actual.Spec.Template)
						assert.NotEmpty(t, actual.Spec.Template.Name)

						if actual.Spec.Type == "clusterresources" {
							assertClusterResourcesTemplate(t, decoder, actual.Spec.Template, expectedTestTiers, tier)
						} else if isNamespaceType(expectedTestTiers, actual.Spec.Type) {
							assertNamespaceTemplate(t, decoder, actual.Spec.Template, expectedTestTiers, tier, actual.Spec.Type)
						} else if isSpaceRole(expectedTestTiers, actual.Spec.Type) {
							assertSpaceRoleTemplate(t, decoder, actual.Spec.Template, expectedTestTiers, tier, actual.Spec.Type)
						} else {
							t.Errorf("unexpected type of template: '%s'", actual.Spec.Type)
						}
					}
				})
			}
		})
	})

	t.Run("failures", func(t *testing.T) {

		t.Run("invalid template", func(t *testing.T) {
			// given
			testTemplates := getTestTemplates(t)
			testTemplates["base/ns_dev.yaml"] = []byte("invalid")
			// when
			_, err := newNSTemplateTierGenerator(s, nil, namespace, getTestMetadata(), testTemplates)

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unable to generate 'advanced-dev-abcd123-123456b' TierTemplate manifest: couldn't get version/kind; json parse error")
		})
	})
}

func ensureObjectFuncForClient(cl runtimeclient.Client) EnsureObject {
	return func(toEnsure runtimeclient.Object, canUpdate bool, _ string) (bool, error) {
		if !canUpdate {
			if err := cl.Create(context.TODO(), toEnsure); err != nil && !apierrors.IsAlreadyExists(err) {
				return false, err
			}
			return true, nil
		}
		applyCl := commonclient.NewApplyClient(cl)
		return applyCl.ApplyObject(context.TODO(), toEnsure, commonclient.ForceUpdate(true))
	}
}

func assertClusterResourcesTemplate(t *testing.T, decoder runtime.Decoder, actual templatev1.Template, expectedTiers map[string]bool, tier string) {
	if !basedOnOtherTier(expectedTiers, tier) {
		expected := templatev1.Template{}
		content := getTestTemplates(t)[(fmt.Sprintf("%s/cluster.yaml", tier))]
		_, _, err := decoder.Decode(content, nil, &expected)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
		assert.NotEmpty(t, actual.Objects)
	}
}

func assertNamespaceTemplate(t *testing.T, decoder runtime.Decoder, actual templatev1.Template, expectedTiers map[string]bool, tier, typeName string) {
	var templatePath string
	if basedOnOtherTier(expectedTiers, tier) {
		templatePath = expectedTemplateFromBasedOnTierConfig(t, tier, fmt.Sprintf("ns_%s.yaml", typeName))
	} else {
		templatePath = fmt.Sprintf("%s/ns_%s.yaml", tier, typeName)
	}
	content := getTestTemplates(t)[templatePath]
	expected := templatev1.Template{}
	_, _, err := decoder.Decode(content, nil, &expected)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
	assert.NotEmpty(t, actual.Objects)
}

func assertSpaceRoleTemplate(t *testing.T, decoder runtime.Decoder, actual templatev1.Template, expectedTiers map[string]bool, tier, roleName string) {
	var templatePath string
	if basedOnOtherTier(expectedTiers, tier) {
		templatePath = expectedTemplateFromBasedOnTierConfig(t, tier, fmt.Sprintf("spacerole_%s.yaml", roleName))
	} else {
		templatePath = fmt.Sprintf("%s/spacerole_%s.yaml", tier, roleName)
	}
	content := getTestTemplates(t)[templatePath]
	expected := templatev1.Template{}
	_, _, err := decoder.Decode(content, nil, &expected)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
	// there are no space role permissions for appstudio-env because the user doesn't have any permissions in the namespace
	if tier != "appstudio-env" {
		assert.NotEmpty(t, actual.Objects)
	} else {
		assert.Empty(t, actual.Objects)
	}
}

func expectedTemplateFromBasedOnTierConfig(t *testing.T, tier, templateFileName string) string {
	basedOnTierContent := getTestTemplates(t)[(fmt.Sprintf("%s/based_on_tier.yaml", tier))]
	basedOnTier := BasedOnTier{}
	require.NoError(t, yaml.Unmarshal(basedOnTierContent, &basedOnTier))
	return fmt.Sprintf("%s/%s", basedOnTier.From, templateFileName)
}

func TestNewNSTemplateTiers(t *testing.T) {

	// given
	s := addToScheme(t)

	t.Run("ok", func(t *testing.T) {
		// given
		namespace := "host-operator-" + uuid.NewString()[:7]
		// when
		tc, err := newNSTemplateTierGenerator(s, nil, namespace, getTestMetadata(), getTestTemplates(t))
		require.NoError(t, err)
		// then
		require.Len(t, tc.templatesByTier, 4)
		for _, name := range []string{"advanced", "base", "nocluster", "appstudio"} {
			tierData, found := tc.templatesByTier[name]
			tierObjs := tierData.objects
			require.Len(t, tierObjs, 1, "expected only 1 NSTemplateTier toolchain object")
			tier := runtimeObjectToNSTemplateTier(t, s, tierObjs[0])

			require.True(t, found)
			assert.Equal(t, name, tier.ObjectMeta.Name)
			assert.Equal(t, namespace, tier.ObjectMeta.Namespace)
			for _, ns := range tier.Spec.Namespaces {
				assert.NotEmpty(t, ns.TemplateRef, "expected namespace reference not empty for tier %v", name)
			}
			if name == "nocluster" {
				assert.Nil(t, tier.Spec.ClusterResources)
			} else {
				require.NotNil(t, tier.Spec.ClusterResources)
				assert.NotEmpty(t, tier.Spec.ClusterResources.TemplateRef)
			}
		}
	})
}

// newNSTemplateTierFromYAML generates toolchainv1alpha1.NSTemplateTier using a golang template which is applied to the given tier.
func newNSTemplateTierFromYAML(s *runtime.Scheme, tier, namespace string, clusterResourcesRevision string, namespaceRevisions map[string]string, spaceRoleRevisions map[string]string) (*toolchainv1alpha1.NSTemplateTier, error) {
	expectedTmpl, err := texttemplate.New("template").Parse(`
{{ $tier := .Tier}}
kind: NSTemplateTier
apiVersion: toolchain.dev.openshift.com/v1alpha1
metadata:
  namespace: {{ .Namespace }}
  name: {{ .Tier }}
spec:
  deactivationTimeoutDays: {{ .DeactivationTimeout }} 
  {{ if .ClusterResourcesRevision }}clusterResources:
    templateRef: {{ .Tier }}-clusterresources-{{ .ClusterResourcesRevision }}
  {{ end }}
  namespaces: 
{{ range $type, $revision := .NamespaceRevisions }}
    - templateRef: {{ $tier }}-{{ $type }}-{{ $revision }}
{{ end }}
  spaceRoles: 
{{ range $role, $revision := .SpaceRoleRevisions }}    
    {{ $role }}:
      templateRef: {{ $tier }}-{{ $role }}-{{ $revision }}
{{ end }}
`)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	err = expectedTmpl.Execute(buf, struct {
		Tier                     string
		Namespace                string
		ClusterResourcesRevision string
		NamespaceRevisions       map[string]string
		SpaceRoleRevisions       map[string]string
		DeactivationTimeout      int
	}{
		Tier:                     tier,
		Namespace:                namespace,
		ClusterResourcesRevision: clusterResourcesRevision,
		NamespaceRevisions:       namespaceRevisions,
		SpaceRoleRevisions:       spaceRoleRevisions,
	})
	if err != nil {
		return nil, err
	}
	result := &toolchainv1alpha1.NSTemplateTier{}
	codecFactory := serializer.NewCodecFactory(s)
	_, _, err = codecFactory.UniversalDeserializer().Decode(buf.Bytes(), nil, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func runtimeObjectToNSTemplateTier(t *testing.T, s *runtime.Scheme, tierObj runtime.Object) *toolchainv1alpha1.NSTemplateTier {
	tier := &toolchainv1alpha1.NSTemplateTier{}
	err := s.Convert(tierObj, tier, nil)
	require.NoError(t, err)
	return tier
}

func addToScheme(t *testing.T) *runtime.Scheme {
	s := scheme.Scheme
	err := toolchainv1alpha1.AddToScheme(s)
	require.NoError(t, err)
	err = templatev1.Install(s)
	require.NoError(t, err)
	return s
}
