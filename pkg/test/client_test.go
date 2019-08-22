package test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	errs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewClient(t *testing.T) {
	fclient := NewFakeClient(t)
	require.NotNil(t, fclient)

	assert.Nil(t, fclient.MockGet)
	assert.Nil(t, fclient.MockList)
	assert.Nil(t, fclient.MockUpdate)
	assert.Nil(t, fclient.MockDelete)
	assert.Nil(t, fclient.MockCreate)
	assert.Nil(t, fclient.MockStatusUpdate)

	key := types.NamespacedName{Namespace: "somenamespace", Name: "somename"}

	t.Run("default methods OK", func(t *testing.T) {
		data := make(map[string]string)
		data["key"] = "value"
		created := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somename",
				Namespace: "somenamespace",
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
		assert.NoError(t, fclient.List(context.TODO(), &client.ListOptions{Namespace: "somenamespace"}, secretList))
		require.Len(t, secretList.Items, 1)
		assert.Equal(t, *created, secretList.Items[0])

		// Update
		created.StringData["key"] = "updated"
		assert.NoError(t, fclient.Update(context.TODO(), created))
		assert.NoError(t, fclient.Get(context.TODO(), key, secret))
		assert.Equal(t, "updated", secret.StringData["key"])

		// Status Update
		assert.NoError(t, fclient.Status().Update(context.TODO(), created))

		// Delete
		assert.NoError(t, fclient.Delete(context.TODO(), created))
		err := fclient.Get(context.TODO(), key, secret)
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
		fclient.MockList = func(ctx context.Context, opts *client.ListOptions, list runtime.Object) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.List(context.TODO(), &client.ListOptions{Namespace: "somenamespace"}, &v1.SecretList{}), expectedErr.Error())
	})

	t.Run("mock Create", func(t *testing.T) {
		defer func() { fclient.MockCreate = nil }()
		fclient.MockCreate = func(ctx context.Context, obj runtime.Object) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.Create(context.TODO(), &v1.Secret{}), expectedErr.Error())
	})

	t.Run("mock Update", func(t *testing.T) {
		defer func() { fclient.MockUpdate = nil }()
		fclient.MockUpdate = func(ctx context.Context, obj runtime.Object) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.Update(context.TODO(), &v1.Secret{}), expectedErr.Error())
	})

	t.Run("mock Delete", func(t *testing.T) {
		defer func() { fclient.MockDelete = nil }()
		fclient.MockDelete = func(ctx context.Context, obj runtime.Object, opts ...client.DeleteOptionFunc) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.Delete(context.TODO(), &v1.Secret{}), expectedErr.Error())
	})

	t.Run("mock Status Update", func(t *testing.T) {
		defer func() { fclient.MockStatusUpdate = nil }()
		fclient.MockStatusUpdate = func(ctx context.Context, obj runtime.Object) error {
			return expectedErr
		}
		assert.EqualError(t, fclient.MockStatusUpdate(context.TODO(), &v1.Secret{}), expectedErr.Error())
	})
}
