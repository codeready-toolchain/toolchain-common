package test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	errs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewClient(t *testing.T) {
	fclient := NewFakeClient(t)
	require.NotNil(t, fclient)

	assert.Nil(t, fclient.MockGet)
	assert.Nil(t, fclient.MockList)
	assert.Nil(t, fclient.MockUpdate)
	assert.Nil(t, fclient.MockPatch)
	assert.Nil(t, fclient.MockDelete)
	assert.Nil(t, fclient.MockDeleteAllOf)
	assert.Nil(t, fclient.MockCreate)
	assert.Nil(t, fclient.MockStatusUpdate)
	assert.Nil(t, fclient.MockStatusPatch)

	key := types.NamespacedName{Namespace: "somenamespace", Name: "somename"}

	t.Run("default methods OK", func(t *testing.T) {
		data := make(map[string]string)
		data["key"] = "value"
		created := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somename",
				Namespace: "somenamespace",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			StringData: data,
		}

		// Create
		assert.NoError(t, fclient.Create(context.TODO(), created))

		// Get
		secret := &v1.Secret{}
		assert.NoError(t, fclient.Get(context.TODO(), key, secret))
		assert.Equal(t, created, secret)

		// List
		secretList := &v1.SecretList{}
		assert.NoError(t, fclient.List(context.TODO(), secretList, client.InNamespace("somenamespace")))
		require.Len(t, secretList.Items, 1)
		assert.Equal(t, *created, secretList.Items[0])

		// Update
		created.StringData["key"] = "updated"
		assert.NoError(t, fclient.Update(context.TODO(), created))
		assert.NoError(t, fclient.Get(context.TODO(), key, secret))
		assert.Equal(t, "updated", secret.StringData["key"])

		// Status Update
		assert.NoError(t, fclient.Status().Update(context.TODO(), created))

		// Patch
		annotations := make(map[string]string)
		annotations["foo"] = "bar"

		mergePatch, err := json.Marshal(map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": annotations,
			},
		})
		require.NoError(t, err)
		assert.NoError(t, fclient.Patch(context.TODO(), created, client.ConstantPatch(types.MergePatchType, mergePatch)))
		assert.NoError(t, fclient.Get(context.TODO(), key, secret))
		assert.Equal(t, annotations, secret.GetObjectMeta().GetAnnotations())

		// Status Patch
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somenamespace",
				Namespace: "somename",
				Labels: map[string]string{
					"foo": "bar",
				},
			}}
		assert.NoError(t, fclient.Create(context.TODO(), dep))
		depPatch := client.MergeFrom(dep.DeepCopy())
		dep.Status.Replicas = 1
		assert.NoError(t, fclient.Status().Patch(context.TODO(), dep, depPatch))

		// Delete
		assert.NoError(t, fclient.Delete(context.TODO(), created))
		err = fclient.Get(context.TODO(), key, secret)
		require.Error(t, err)
		assert.True(t, errs.IsNotFound(err))

		// DeleteAllOf
		dep2 := dep.DeepCopy()
		dep2.Name = dep2.Name + "-2"
		assert.NoError(t, fclient.Create(context.TODO(), dep2))

		assert.NoError(t, fclient.DeleteAllOf(context.TODO(), dep, client.InNamespace("somenamespace"), client.MatchingLabels(dep.ObjectMeta.Labels)))
		err = fclient.Get(context.TODO(), key, dep)
		require.Error(t, err)
		assert.True(t, errs.IsNotFound(err))

		err = fclient.Get(context.TODO(), key, dep2)
		require.Error(t, err)
		assert.True(t, errs.IsNotFound(err))
	})

	expectedErr := errors.New("oopsie woopsie")

	t.Run("mock Get", func(t *testing.T) {
		defer func() { fclient.MockGet = nil }()
		fclient.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.Get(context.TODO(), key, &v1.Secret{}), expectedErr.Error())
	})

	t.Run("mock List", func(t *testing.T) {
		defer func() { fclient.MockList = nil }()
		fclient.MockList = func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.List(context.TODO(), &v1.SecretList{}, client.InNamespace("somenamespace")), expectedErr.Error())
	})

	t.Run("mock Create", func(t *testing.T) {
		defer func() { fclient.MockCreate = nil }()
		fclient.MockCreate = func(ctx context.Context, obj runtime.Object, option ...client.CreateOption) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.Create(context.TODO(), &v1.Secret{}), expectedErr.Error())
	})

	t.Run("mock Update", func(t *testing.T) {
		defer func() { fclient.MockUpdate = nil }()
		fclient.MockUpdate = func(ctx context.Context, obj runtime.Object, option ...client.UpdateOption) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.Update(context.TODO(), &v1.Secret{}), expectedErr.Error())
	})

	t.Run("mock Patch", func(t *testing.T) {
		defer func() { fclient.MockPatch = nil }()
		fclient.MockPatch = func(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.Patch(context.TODO(), &v1.Secret{}, client.ConstantPatch(types.MergePatchType, []byte{})), expectedErr.Error())
	})

	t.Run("mock Delete", func(t *testing.T) {
		defer func() { fclient.MockDelete = nil }()
		fclient.MockDelete = func(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.Delete(context.TODO(), &v1.Secret{}), expectedErr.Error())
	})

	t.Run("mock DeleteAllOf", func(t *testing.T) {
		defer func() { fclient.MockDeleteAllOf = nil }()
		fclient.MockDeleteAllOf = func(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.DeleteAllOf(context.TODO(), &v1.Secret{}), expectedErr.Error())
	})

	t.Run("mock Status Update", func(t *testing.T) {
		defer func() { fclient.MockStatusUpdate = nil }()
		fclient.MockStatusUpdate = func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.MockStatusUpdate(context.TODO(), &v1.Secret{}), expectedErr.Error())
	})

	t.Run("mock Status Patch", func(t *testing.T) {
		defer func() { fclient.MockStatusPatch = nil }()
		fclient.MockStatusPatch = func(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.MockStatusPatch(context.TODO(), &v1.Secret{}, client.ConstantPatch(types.MergePatchType, []byte{})), expectedErr.Error())
	})
}
