package masteruserrecord_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/codeready-toolchain/api/pkg/apis"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/codeready-toolchain/toolchain-common/pkg/test/masteruserrecord"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestMasterUserRecordAssertion(t *testing.T) {

	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)

	t.Run("HasNSTemplateSet assertion", func(t *testing.T) {

		mur := masteruserrecord.NewMasterUserRecord("foo", masteruserrecord.TargetCluster("cluster-1"))

		t.Run("ok", func(t *testing.T) {
			// given
			mockT := NewMockT()
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
			a := masteruserrecord.AssertThatMasterUserRecord(mockT, "foo", client)
			expectedTmplSet := toolchainv1alpha1.NSTemplateSetSpec{
				TierName: "basic",
				Namespaces: []toolchainv1alpha1.NSTemplateSetNamespace{
					{
						Type:     "dev",
						Revision: "123abc",
						Template: "",
					},
					{
						Type:     "code",
						Revision: "123abc",
						Template: "",
					},
					{
						Type:     "stage",
						Revision: "123abc",
						Template: "",
					},
				},
				ClusterResources: &toolchainv1alpha1.NSTemplateSetClusterResources{
					Revision: "654321a",
				},
			}
			// when
			a.HasNSTemplateSet("cluster-1", expectedTmplSet)
			// then: all good
			assert.False(t, mockT.CalledFailNow())
			assert.False(t, mockT.CalledFatalf())
			assert.False(t, mockT.CalledErrorf())
		})

		t.Run("failures", func(t *testing.T) {

			t.Run("missing target cluster", func(t *testing.T) {
				// given
				mockT := NewMockT()
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
				a := masteruserrecord.AssertThatMasterUserRecord(mockT, "foo", client)
				expectedTmplSet := toolchainv1alpha1.NSTemplateSetSpec{
					TierName: "basic",
					Namespaces: []toolchainv1alpha1.NSTemplateSetNamespace{
						{
							Type:     "dev",
							Revision: "123abc",
							Template: "",
						},
						{
							Type:     "code",
							Revision: "123abc",
							Template: "",
						},
						{
							Type:     "stage",
							Revision: "123abc",
							Template: "",
						},
					},
					ClusterResources: &toolchainv1alpha1.NSTemplateSetClusterResources{
						Revision: "654321a",
					},
				}
				// when
				a.HasNSTemplateSet("cluster-unknown", expectedTmplSet)
				// then
				assert.False(t, mockT.CalledFailNow())
				assert.False(t, mockT.CalledErrorf())
				assert.True(t, mockT.CalledFatalf()) // no match found for the given cluster
			})

			t.Run("different NSTemplateSets", func(t *testing.T) {
				// given
				mockT := NewMockT()
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
				a := masteruserrecord.AssertThatMasterUserRecord(mockT, "foo", client)
				expectedTmplSet := toolchainv1alpha1.NSTemplateSetSpec{
					TierName:         "basic",
					Namespaces:       []toolchainv1alpha1.NSTemplateSetNamespace{},
					ClusterResources: nil,
				}
				// when
				a.HasNSTemplateSet("cluster-1", expectedTmplSet)
				// then
				assert.False(t, mockT.CalledFailNow())
				assert.False(t, mockT.CalledFatalf())
				assert.True(t, mockT.CalledErrorf()) // assert.Equal failed
			})
		})
	})
}

func NewMockT() *MockT {
	return &MockT{}
}

type MockT struct {
	logfCount    int
	errorfCount  int
	fatalfCount  int
	failnowCount int
}

func (t *MockT) Logf(format string, args ...interface{}) {
	t.logfCount++
}

func (t *MockT) Errorf(format string, args ...interface{}) {
	t.errorfCount++
}

func (t *MockT) Fatalf(format string, args ...interface{}) {
	t.fatalfCount++

}

func (t *MockT) FailNow() {
	t.failnowCount++
}

func (t *MockT) CalledLogf() bool {
	return t.logfCount > 0
}

func (t *MockT) CalledErrorf() bool {
	return t.errorfCount > 0
}

func (t *MockT) CalledFatalf() bool {
	return t.fatalfCount > 0
}

func (t *MockT) CalledFailNow() bool {
	return t.failnowCount > 0
}
