package masteruserrecord_test

import (
	"context"
	"fmt"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	murtest "github.com/codeready-toolchain/toolchain-common/pkg/test/masteruserrecord"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMasterUserRecordAssertion(t *testing.T) {

	s := scheme.Scheme
	err := toolchainv1alpha1.AddToScheme(s)
	require.NoError(t, err)

	t.Run("HasTier assertion", func(t *testing.T) {

		mur := murtest.NewMasterUserRecord(t, "foo", murtest.TargetCluster(test.MemberClusterName))

		t.Run("ok", func(t *testing.T) {
			// given
			mockT := test.NewMockT()
			client := test.NewFakeClient(mockT, mur)
			client.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtimeclient.Object) error {
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
				HasTier(murtest.DefaultNSTemplateTier())
			// then: all good
			assert.False(t, mockT.CalledFailNow())
			assert.False(t, mockT.CalledFatalf())
			assert.False(t, mockT.CalledErrorf())
		})

		t.Run("failures", func(t *testing.T) {

			t.Run("does not have matching tier", func(t *testing.T) {
				// given
				mockT := test.NewMockT()
				client := test.NewFakeClient(mockT, mur)
				client.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtimeclient.Object) error {
					if key.Namespace == test.HostOperatorNs && key.Name == "foo" {
						if obj, ok := obj.(*toolchainv1alpha1.MasterUserRecord); ok {
							*obj = *mur
							return nil
						}
					}
					return fmt.Errorf("unexpected object key: %v", key)
				}
				otherTier := murtest.DefaultNSTemplateTier()
				otherTier.Name = "other"
				// when
				murtest.AssertThatMasterUserRecord(mockT, "foo", client).
					HasTier(otherTier)
				// then
				assert.False(t, mockT.CalledFailNow())
				assert.True(t, mockT.CalledErrorf()) // no match found for the given tier
				assert.False(t, mockT.CalledFatalf())
			})

		})
	})
}

func TestHasAnnotationWithValue(t *testing.T) {

	t.Run("should succeed when annotation with value exists", func(t *testing.T) {
		// given
		mur := murtest.NewMasterUserRecord(t, "john", murtest.WithAnnotation("toolchain/email", "john@example.com"))
		mockT := test.NewMockT()
		client := test.NewFakeClient(mockT, mur)

		// when
		murtest.AssertThatMasterUserRecord(mockT, "john", client).HasAnnotationWithValue("toolchain/email", "john@example.com")

		// then: all good
		assert.False(t, mockT.CalledFailNow())
		assert.False(t, mockT.CalledFatalf())
		assert.False(t, mockT.CalledErrorf())
	})

	t.Run("should fail when there is no annotations", func(t *testing.T) {
		// given
		mur := murtest.NewMasterUserRecord(t, "john")
		mockT := test.NewMockT()
		client := test.NewFakeClient(mockT, mur)

		// when
		murtest.AssertThatMasterUserRecord(mockT, "john", client).HasAnnotationWithValue("toolchain/email", "john@example.com")

		// then expect an error and test failure
		assert.True(t, mockT.CalledFailNow())
		assert.False(t, mockT.CalledFatalf())
		assert.True(t, mockT.CalledErrorf())
	})

	t.Run("should fail when annotation does not exist", func(t *testing.T) {
		// given
		mur := murtest.NewMasterUserRecord(t, "john", murtest.WithAnnotation("other/stuff", "whatever"))
		mockT := test.NewMockT()
		client := test.NewFakeClient(mockT, mur)

		// when
		murtest.AssertThatMasterUserRecord(mockT, "john", client).HasAnnotationWithValue("toolchain/email", "john@example.com")

		// then expect an error and test failure
		assert.True(t, mockT.CalledFailNow())
		assert.False(t, mockT.CalledFatalf())
		assert.True(t, mockT.CalledErrorf())
	})

	t.Run("should fail when annotation exists with different value", func(t *testing.T) {
		// given
		mur := murtest.NewMasterUserRecord(t, "john", murtest.WithAnnotation("toolchain/email", "john@example.com"))
		mockT := test.NewMockT()
		client := test.NewFakeClient(mockT, mur)

		// when
		murtest.AssertThatMasterUserRecord(mockT, "john", client).HasAnnotationWithValue("toolchain/email", "other@example.com")

		// then expect an error
		assert.False(t, mockT.CalledFailNow())
		assert.False(t, mockT.CalledFatalf())
		assert.True(t, mockT.CalledErrorf())
	})
}
