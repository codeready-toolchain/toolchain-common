package masteruserrecord_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/codeready-toolchain/api/pkg/apis"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	murtest "github.com/codeready-toolchain/toolchain-common/pkg/test/masteruserrecord"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestMasterUserRecordAssertion(t *testing.T) {

	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)

	t.Run("HasNSTemplateSet assertion", func(t *testing.T) {

		mur := murtest.NewMasterUserRecord(t, "foo", murtest.TargetCluster(test.MemberClusterName))

		t.Run("ok", func(t *testing.T) {
			// given
			mockT := test.NewMockT()
			client := test.NewFakeClient(mockT, mur)
			client.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
				if key.Namespace == test.HostOperatorNs && key.Name == "foo" {
					if obj, ok := obj.(*toolchainv1alpha1.MasterUserRecord); ok {
						*obj = *mur
						return nil
					}
				}
				return fmt.Errorf("unexpected object key: %v", key)
			}
			// when
			murtest.AssertThatMasterUserRecord(mockT, "foo", client).
				HasNSTemplateSet(test.MemberClusterName,
					murtest.WithTier("basic"),
					murtest.WithNs("dev", "123abc"),
					murtest.WithNs("code", "123abc"),
					murtest.WithNs("stage", "123abc"),
					murtest.WithClusterRes("654321a"))
			// then: all good
			assert.False(t, mockT.CalledFailNow())
			assert.False(t, mockT.CalledFatalf())
			assert.False(t, mockT.CalledErrorf())
		})

		t.Run("failures", func(t *testing.T) {

			t.Run("missing target cluster", func(t *testing.T) {
				// given
				mockT := test.NewMockT()
				client := test.NewFakeClient(mockT, mur)
				client.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					if key.Namespace == test.HostOperatorNs && key.Name == "foo" {
						if obj, ok := obj.(*toolchainv1alpha1.MasterUserRecord); ok {
							*obj = *mur
							return nil
						}
					}
					return fmt.Errorf("unexpected object key: %v", key)
				}
				// when
				murtest.AssertThatMasterUserRecord(mockT, "foo", client).
					HasNSTemplateSet("cluster-unknown",
						murtest.WithTier("basic"),
						murtest.WithNs("dev", "123abc"),
						murtest.WithNs("code", "123abc"),
						murtest.WithNs("stage", "123abc"),
						murtest.WithClusterRes("654321a"))
				// then
				assert.False(t, mockT.CalledFailNow())
				assert.False(t, mockT.CalledErrorf())
				assert.True(t, mockT.CalledFatalf()) // no match found for the given cluster
			})

			t.Run("different NSTemplateSets", func(t *testing.T) {
				// given
				mockT := test.NewMockT()
				client := test.NewFakeClient(mockT, mur)
				client.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					if key.Namespace == test.HostOperatorNs && key.Name == "foo" {
						if obj, ok := obj.(*toolchainv1alpha1.MasterUserRecord); ok {
							*obj = *mur
							return nil
						}
					}
					return fmt.Errorf("unexpected object key: %v", key)
				}
				// when
				murtest.AssertThatMasterUserRecord(mockT, "foo", client).
					HasNSTemplateSet(test.MemberClusterName, murtest.WithTier("basic"))
				// then
				assert.False(t, mockT.CalledFailNow())
				assert.False(t, mockT.CalledFatalf())
				assert.True(t, mockT.CalledErrorf()) // assert.Equal failed
			})
		})
	})

	t.Run("UserAccountHasTier assertion", func(t *testing.T) {

		mur := murtest.NewMasterUserRecord(t, "foo", murtest.TargetCluster(test.MemberClusterName))

		t.Run("ok", func(t *testing.T) {
			// given
			tier := toolchainv1alpha1.NSTemplateTier{
				ObjectMeta: v1.ObjectMeta{
					Name: "basic",
				},
				Spec: toolchainv1alpha1.NSTemplateTierSpec{
					Namespaces: []toolchainv1alpha1.NSTemplateTierNamespace{
						{
							TemplateRef: "basic-dev-123abc",
						},
						{
							TemplateRef: "basic-code-123abc",
						},
						{
							TemplateRef: "basic-stage-123abc",
						},
					},
					ClusterResources: &toolchainv1alpha1.NSTemplateTierClusterResources{
						TemplateRef: "basic-clusterresources-654321a",
					},
				},
			}
			mockT := test.NewMockT()
			client := test.NewFakeClient(mockT, mur)
			client.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
				if key.Namespace == test.HostOperatorNs && key.Name == "foo" {
					if obj, ok := obj.(*toolchainv1alpha1.MasterUserRecord); ok {
						*obj = *mur
						return nil
					}
				}
				return fmt.Errorf("unexpected object key: %v", key)
			}
			// when
			murtest.AssertThatMasterUserRecord(mockT, "foo", client).
				UserAccountHasTier(test.MemberClusterName, tier)
			// then: all good
			assert.False(t, mockT.CalledFailNow())
			assert.False(t, mockT.CalledFatalf())
			assert.False(t, mockT.CalledErrorf())
		})

		t.Run("failures", func(t *testing.T) {

			t.Run("missing stage namespace", func(t *testing.T) {
				// given
				tier := toolchainv1alpha1.NSTemplateTier{
					ObjectMeta: v1.ObjectMeta{
						Name: "basic",
					},
					Spec: toolchainv1alpha1.NSTemplateTierSpec{
						Namespaces: []toolchainv1alpha1.NSTemplateTierNamespace{
							{
								TemplateRef: "basic-dev-123abc",
							},
							{
								TemplateRef: "basic-code-123abc",
							},
						},
					},
				}
				mockT := test.NewMockT()
				client := test.NewFakeClient(mockT, mur)
				client.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					if key.Namespace == test.HostOperatorNs && key.Name == "foo" {
						if obj, ok := obj.(*toolchainv1alpha1.MasterUserRecord); ok {
							*obj = *mur
							return nil
						}
					}
					return fmt.Errorf("unexpected object key: %v", key)
				}
				// when
				murtest.AssertThatMasterUserRecord(mockT, "foo", client).
					UserAccountHasTier(test.MemberClusterName, tier)
				// then: all good
				assert.False(t, mockT.CalledFailNow())
				assert.False(t, mockT.CalledFatalf())
				assert.True(t, mockT.CalledErrorf()) // assert.Equal failed
			})

			t.Run("invalid stage namespace", func(t *testing.T) {
				// given
				tier := toolchainv1alpha1.NSTemplateTier{
					ObjectMeta: v1.ObjectMeta{
						Name: "basic",
					},
					Spec: toolchainv1alpha1.NSTemplateTierSpec{
						Namespaces: []toolchainv1alpha1.NSTemplateTierNamespace{
							{
								TemplateRef: "basic-dev-123abc",
							},
							{
								TemplateRef: "basic-code-123abc",
							},
							{
								TemplateRef: "basic-stage-invalid",
							},
						},
					},
				}
				mockT := test.NewMockT()
				client := test.NewFakeClient(mockT, mur)
				client.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					if key.Namespace == test.HostOperatorNs && key.Name == "foo" {
						if obj, ok := obj.(*toolchainv1alpha1.MasterUserRecord); ok {
							*obj = *mur
							return nil
						}
					}
					return fmt.Errorf("unexpected object key: %v", key)
				}
				// when
				murtest.AssertThatMasterUserRecord(mockT, "foo", client).
					UserAccountHasTier(test.MemberClusterName, tier)
				// then: all good
				assert.False(t, mockT.CalledFailNow())
				assert.False(t, mockT.CalledFatalf())
				assert.True(t, mockT.CalledErrorf()) // assert.Equal failed
			})

			t.Run("missing cluster resources", func(t *testing.T) {
				// given
				tier := toolchainv1alpha1.NSTemplateTier{
					ObjectMeta: v1.ObjectMeta{
						Name: "basic",
					},
					Spec: toolchainv1alpha1.NSTemplateTierSpec{
						Namespaces: []toolchainv1alpha1.NSTemplateTierNamespace{
							{
								TemplateRef: "basic-dev-123abc",
							},
							{
								TemplateRef: "basic-code-123abc",
							},
							{
								TemplateRef: "basic-stage-123abc",
							},
						},
					},
				}
				mockT := test.NewMockT()
				client := test.NewFakeClient(mockT, mur)
				client.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					if key.Namespace == test.HostOperatorNs && key.Name == "foo" {
						if obj, ok := obj.(*toolchainv1alpha1.MasterUserRecord); ok {
							*obj = *mur
							return nil
						}
					}
					return fmt.Errorf("unexpected object key: %v", key)
				}
				// when
				murtest.AssertThatMasterUserRecord(mockT, "foo", client).
					UserAccountHasTier(test.MemberClusterName, tier)
				// then: all good
				assert.False(t, mockT.CalledFailNow())
				assert.False(t, mockT.CalledFatalf())
				assert.True(t, mockT.CalledErrorf()) // assert.Equal failed
			})

			t.Run("invalid cluster resources", func(t *testing.T) {
				// given
				tier := toolchainv1alpha1.NSTemplateTier{
					ObjectMeta: v1.ObjectMeta{
						Name: "basic",
					},
					Spec: toolchainv1alpha1.NSTemplateTierSpec{
						Namespaces: []toolchainv1alpha1.NSTemplateTierNamespace{
							{
								TemplateRef: "basic-dev-123abc",
							},
							{
								TemplateRef: "basic-code-123abc",
							},
							{
								TemplateRef: "basic-stage-123abc",
							},
						},
						ClusterResources: &toolchainv1alpha1.NSTemplateTierClusterResources{
							TemplateRef: "invalid",
						},
					},
				}
				mockT := test.NewMockT()
				client := test.NewFakeClient(mockT, mur)
				client.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					if key.Namespace == test.HostOperatorNs && key.Name == "foo" {
						if obj, ok := obj.(*toolchainv1alpha1.MasterUserRecord); ok {
							*obj = *mur
							return nil
						}
					}
					return fmt.Errorf("unexpected object key: %v", key)
				}
				// when
				murtest.AssertThatMasterUserRecord(mockT, "foo", client).
					UserAccountHasTier(test.MemberClusterName, tier)
				// then: all good
				assert.False(t, mockT.CalledFailNow())
				assert.False(t, mockT.CalledFatalf())
				assert.True(t, mockT.CalledErrorf()) // assert.Equal failed
			})
		})

	})
}
