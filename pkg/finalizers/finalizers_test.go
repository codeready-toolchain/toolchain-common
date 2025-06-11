package finalizers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/finalizer"
)

func TestFinalizers(t *testing.T) {
	t.Run("adds a finalizer on non-deleted", func(t *testing.T) {
		// given
		var fs Finalizers
		require.NoError(t, fs.Register("dummy", FinalizerFunc(func(ctx context.Context, o client.Object) (finalizer.Result, error) {
			return finalizer.Result{}, nil
		})))
		obj := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cm",
				Namespace: test.HostOperatorNs,
			},
		}
		cl := fake.NewClientBuilder().WithObjects(obj).Build()

		// when
		updated, err := fs.FinalizeAndUpdate(context.TODO(), cl, obj)
		inCluster := &corev1.ConfigMap{}
		require.NoError(t, cl.Get(context.TODO(), client.ObjectKeyFromObject(obj), inCluster))

		// then
		require.NoError(t, err)
		assert.True(t, updated)
		assert.Contains(t, inCluster.Finalizers, "dummy")
	})

	t.Run("does not modify when finalizer already present", func(t *testing.T) {
		// given
		var fs Finalizers
		require.NoError(t, fs.Register("dummy", FinalizerFunc(func(ctx context.Context, o client.Object) (finalizer.Result, error) {
			return finalizer.Result{}, nil
		})))
		obj := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "cm",
				Namespace:  test.HostOperatorNs,
				Finalizers: []string{"dummy"},
			},
		}
		cl := fake.NewClientBuilder().WithObjects(obj).Build()

		// when
		updated, err := fs.FinalizeAndUpdate(context.TODO(), cl, obj)
		inCluster := &corev1.ConfigMap{}
		require.NoError(t, cl.Get(context.TODO(), client.ObjectKeyFromObject(obj), inCluster))

		// then
		require.NoError(t, err)
		assert.False(t, updated)
		assert.Contains(t, inCluster.Finalizers, "dummy")
	})

	t.Run("removes the finalizer when it runs successfully on deleted object", func(t *testing.T) {
		// given
		var fs Finalizers

		require.NoError(t, fs.Register("dummy", FinalizerFunc(func(ctx context.Context, o client.Object) (finalizer.Result, error) {
			return finalizer.Result{}, nil
		})))
		obj := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "cm",
				Namespace:         test.HostOperatorNs,
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
				Finalizers:        []string{"dummy", "other"},
			},
		}
		cl := fake.NewClientBuilder().WithObjects(obj).Build()

		// when
		updated, err := fs.FinalizeAndUpdate(context.TODO(), cl, obj)
		inCluster := &corev1.ConfigMap{}
		require.NoError(t, cl.Get(context.TODO(), client.ObjectKeyFromObject(obj), inCluster))

		// then
		require.NoError(t, err)
		assert.True(t, updated)
		assert.Len(t, inCluster.Finalizers, 1)
		assert.Contains(t, inCluster.Finalizers, "other")
	})

	t.Run("updates even on error", func(t *testing.T) {
		// given
		var fs Finalizers

		require.NoError(t, fs.Register("dummy", FinalizerFunc(func(ctx context.Context, o client.Object) (finalizer.Result, error) {
			cm := o.(*corev1.ConfigMap)
			cm.Data = map[string]string{"key": "value"}

			return finalizer.Result{Updated: true}, fmt.Errorf("intentional error")
		})))
		obj := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "cm",
				Namespace:         test.HostOperatorNs,
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
				Finalizers:        []string{"dummy"},
			},
		}
		cl := fake.NewClientBuilder().WithObjects(obj).Build()

		// when
		updated, err := fs.FinalizeAndUpdate(context.TODO(), cl, obj)
		inCluster := &corev1.ConfigMap{}
		require.NoError(t, cl.Get(context.TODO(), client.ObjectKeyFromObject(obj), inCluster))

		// then
		require.Error(t, err)
		assert.True(t, updated)
		assert.Equal(t, "value", inCluster.Data["key"])
	})
}
