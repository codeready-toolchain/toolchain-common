package client_test

import (
	"context"
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSsaClient(t *testing.T) {
	t.Run("ApplyObject", func(t *testing.T) {
		t.Run("creates", func(t *testing.T) {
			// given
			cl, acl := NewTestSsaApplyClient(t)
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj",
					Namespace: "default",
				},
			}

			// when
			_, err := acl.ApplyObject(context.TODO(), obj)
			require.NoError(t, err)

			// then
			inCluster := &corev1.ConfigMap{}
			require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))
		})
		t.Run("updates", func(t *testing.T) {
			// given
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj",
					Namespace: "default",
				},
				Data: map[string]string{"a": "b"},
			}
			cl, acl := NewTestSsaApplyClient(t, obj)

			updated := obj.DeepCopy()
			updated.Data["a"] = "c"

			// when
			_, err := acl.ApplyObject(context.TODO(), updated)
			require.NoError(t, err)
			inCluster := &corev1.ConfigMap{}
			require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))

			// then
			assert.Equal(t, "c", inCluster.Data["a"])
		})
		t.Run("SetOwner", func(t *testing.T) {
			// given
			owner := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owner",
					Namespace: "default",
				},
			}

			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owned",
					Namespace: "default",
				},
			}
			cl, acl := NewTestSsaApplyClient(t, owner, obj)

			// when
			_, err := acl.ApplyObject(context.TODO(), obj, client.SsaSetOwner(owner))
			require.NoError(t, err)
			inCluster := &corev1.ConfigMap{}
			require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))

			// then
			require.Len(t, inCluster.OwnerReferences, 1)
			assert.Equal(t, "ConfigMap", inCluster.OwnerReferences[0].Kind)
			assert.Equal(t, "owner", inCluster.OwnerReferences[0].Name)
		})
		t.Run("EnsureLabels", func(t *testing.T) {
			// given
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj",
					Namespace: "default",
				},
			}
			cl, acl := NewTestSsaApplyClient(t, obj)

			// when
			_, err := acl.ApplyObject(context.TODO(), obj, client.EnsureLabels(map[string]string{"a": "b", "c": "d"}))
			require.NoError(t, err)
			inCluster := &corev1.ConfigMap{}
			require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))

			// then
			require.NotNil(t, inCluster.Labels)
			require.Len(t, inCluster.Labels, 2)
			assert.Equal(t, "b", inCluster.Labels["a"])
			assert.Equal(t, "d", inCluster.Labels["c"])
		})
		t.Run("SkipIf", func(t *testing.T) {
			// given
			cl, acl := NewTestSsaApplyClient(t)
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj",
					Namespace: "default",
				},
			}

			// when
			_, err := acl.ApplyObject(context.TODO(), obj, client.SkipIf(func(o runtimeclient.Object) bool {
				return true
			}))
			require.NoError(t, err)

			// then
			inCluster := &corev1.ConfigMap{}
			require.True(t, errors.IsNotFound(cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster)))
		})
		t.Run("DetermineUpdate", func(t *testing.T) {
			obj := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj",
					Namespace: "default",
				},
			}
			_, acl := NewTestSsaApplyClient(t, obj.DeepCopy())

			// NOTE: the order of the tests IS SIGNIFICANT - we're doing the same thing twice and checking that no update happens
			// the second time.

			t.Run("detects change", func(t *testing.T) {
				// given
				updatedObj := obj.DeepCopy()
				updatedObj.Spec.Selector = map[string]string{"a": "b"}

				// when
				wasUpdated, err := acl.ApplyObject(context.TODO(), updatedObj, client.DetermineUpdate(true))
				require.NoError(t, err)

				// then
				assert.True(t, wasUpdated)
			})
			t.Run("detects no change", func(t *testing.T) {
				// given
				updatedObj := obj.DeepCopy()
				updatedObj.Spec.Selector = map[string]string{"a": "b"}

				// when
				wasUpdated, err := acl.ApplyObject(context.TODO(), updatedObj, client.DetermineUpdate(true))
				require.NoError(t, err)

				// then
				assert.False(t, wasUpdated)
			})
		})
	})
}

func NewTestSsaApplyClient(t *testing.T, initObjs ...runtimeclient.Object) (runtimeclient.Client, *client.SsaApplyClient) {
	cl := test.NewFakeClient(t, initObjs...)
	test.FakeSSA(cl)

	return cl, client.NewSsaApplyClient(cl)
}
