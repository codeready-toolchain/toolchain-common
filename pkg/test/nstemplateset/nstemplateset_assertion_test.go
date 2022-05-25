package nstemplateset_test

import (
	"context"
	"fmt"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/codeready-toolchain/toolchain-common/pkg/test/nstemplateset"
	"k8s.io/apimachinery/pkg/types"

	"github.com/stretchr/testify/assert"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNSTemplateSetAssertion(t *testing.T) {

	t.Run("check space roles", func(t *testing.T) {
		// given
		nsTmplSet := nstemplateset.NewNSTemplateSet("foo")
		nsTmplSet.Spec.SpaceRoles = []toolchainv1alpha1.NSTemplateSetSpaceRole{
			{
				TemplateRef: "base-admin-123456b",
				Usernames:   []string{"admin1", "admin2"},
			},
			{
				TemplateRef: "base-viewer-123456b",
				Usernames:   []string{"viewer1"},
			},
		}
		cl := test.NewFakeClient(t, nsTmplSet)
		cl.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtimeclient.Object) error {
			if key.Namespace == test.MemberOperatorNs && key.Name == "foo" {
				if obj, ok := obj.(*toolchainv1alpha1.NSTemplateSet); ok {
					*obj = *nsTmplSet
					return nil
				}
			}
			return fmt.Errorf("unexpected object key: %v", key)
		}

		t.Run("success", func(t *testing.T) {
			// given
			mockT := test.NewMockT()
			// when
			nstemplateset.AssertThatNSTemplateSet(mockT, nsTmplSet.Namespace, nsTmplSet.Name, cl).HasSpaceRoles(
				nstemplateset.SpaceRole("base-admin-123456b", "admin1", "admin2"),
				nstemplateset.SpaceRole("base-viewer-123456b", "viewer1"),
			)
			// then: all good
			assert.False(t, mockT.CalledFailNow())
			assert.False(t, mockT.CalledFatalf())
			assert.False(t, mockT.CalledErrorf()) // called
		})

		t.Run("fails when actual user is not expected", func(t *testing.T) {
			// given
			mockT := test.NewMockT()
			// when
			nstemplateset.AssertThatNSTemplateSet(mockT, test.MemberOperatorNs, "foo", cl).HasSpaceRoles(
				nstemplateset.SpaceRole("base-admin-123456b", "admin1"), // admin2 is not expected
				nstemplateset.SpaceRole("base-viewer-123456b", "viewer1"),
			)
			// then: all good
			assert.False(t, mockT.CalledFailNow())
			assert.False(t, mockT.CalledFatalf())
			assert.True(t, mockT.CalledErrorf()) // called
		})

		t.Run("fails when expected user is missing in actual", func(t *testing.T) {
			// given
			mockT := test.NewMockT()
			// when
			nstemplateset.AssertThatNSTemplateSet(mockT, test.MemberOperatorNs, "foo", cl).HasSpaceRoles(
				nstemplateset.SpaceRole("base-admin-123456b", "admin1", "admin2", "admin3"), // admin3 is expected but missing
				nstemplateset.SpaceRole("base-viewer-123456b", "viewer1"),
			)
			// then: all good
			assert.False(t, mockT.CalledFailNow())
			assert.False(t, mockT.CalledFatalf())
			assert.True(t, mockT.CalledErrorf()) // called
		})

		t.Run("fails when actual role is not expected", func(t *testing.T) {
			// given
			mockT := test.NewMockT()
			// when
			nstemplateset.AssertThatNSTemplateSet(mockT, test.MemberOperatorNs, "foo", cl).HasSpaceRoles(
				nstemplateset.SpaceRole("base-admin-123456b", "admin1", "admin2"),
				//  missing `base-viewer-123456b` with `viewer1`
			)
			// then: all good
			assert.False(t, mockT.CalledFailNow())
			assert.False(t, mockT.CalledFatalf())
			assert.True(t, mockT.CalledErrorf()) // called
		})

		t.Run("fails when expected role is missing", func(t *testing.T) {
			// given
			mockT := test.NewMockT()
			// when
			nstemplateset.AssertThatNSTemplateSet(mockT, test.MemberOperatorNs, "foo", cl).HasSpaceRoles(
				nstemplateset.SpaceRole("base-admin-123456b", "admin1", "admin2"),
				nstemplateset.SpaceRole("base-viewer-123456b", "viewer1"),
				nstemplateset.SpaceRole("base-other-123456b", "other1"),
			)
			// then: all good
			assert.False(t, mockT.CalledFailNow())
			assert.False(t, mockT.CalledFatalf())
			assert.True(t, mockT.CalledErrorf()) // called
		})
	})

}
