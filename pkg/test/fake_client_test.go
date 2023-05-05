package test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	errs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestNewClient(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	cl := NewFakeClient(t)
	require.NotNil(t, cl)

	assert.Nil(t, cl.MockGet)
	assert.Nil(t, cl.MockList)
	assert.Nil(t, cl.MockUpdate)
	assert.Nil(t, cl.MockPatch)
	assert.Nil(t, cl.MockDelete)
	// assert.Nil(t, fclient.MockDeleteAllOf)
	assert.Nil(t, cl.MockCreate)
	assert.Nil(t, cl.MockStatusUpdate)
	assert.Nil(t, cl.MockStatusPatch)

	t.Run("default methods OK", func(t *testing.T) {
		t.Run("list", func(t *testing.T) {
			created, _ := createAndGetSecret(t, logger, cl)
			secretList := &v1.SecretList{}
			assert.NoError(t, cl.List(context.TODO(), secretList, client.InNamespace("somenamespace")))
			require.Len(t, secretList.Items, 1)
			assert.Equal(t, *created, secretList.Items[0])
		})

		t.Run("update object with stringData", func(t *testing.T) {
			created, retrieved := createAndGetSecret(t, logger, cl)
			created.StringData["key"] = "updated"
			assert.NoError(t, cl.Update(context.TODO(), logger, created))
			assert.NoError(t, cl.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			assert.Equal(t, "updated", retrieved.StringData["key"])
			assert.EqualValues(t, 2, retrieved.Generation) // Generation updated
		})

		t.Run("update object with the same stringData", func(t *testing.T) {
			created, retrieved := createAndGetSecret(t, logger, cl)
			assert.NoError(t, cl.Update(context.TODO(), logger, created))
			assert.NoError(t, cl.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			assert.Equal(t, "value", retrieved.StringData["key"])
			assert.EqualValues(t, 1, retrieved.Generation) // Generation updated
		})

		t.Run("update object with data", func(t *testing.T) {
			key := types.NamespacedName{Namespace: "somenamespace", Name: "somename" + uuid.Must(uuid.NewV4()).String()}
			data := make(map[string][]byte)
			data["key"] = []byte("value")
			created := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				Data: data,
			}
			// Create
			assert.NoError(t, cl.Create(context.TODO(), logger, created))
			// Get
			secret := &v1.Secret{}
			assert.NoError(t, cl.Get(context.TODO(), key, secret))
			assert.Equal(t, created, secret)
			assert.EqualValues(t, 1, secret.Generation)

			data["newkey"] = []byte("newvalue")
			created.Data = data
			assert.NoError(t, cl.Update(context.TODO(), logger, created))
			assert.NoError(t, cl.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, secret))
			assert.Equal(t, []byte("value"), secret.Data["key"])
			assert.Equal(t, []byte("newvalue"), secret.Data["newkey"])
			assert.EqualValues(t, 2, secret.Generation) // Generation updated
		})

		t.Run("update object with spec", func(t *testing.T) {
			created, retrieved := createAndGetDeployment(t, logger, cl)
			newReplicas := int32(10)
			created.Spec.Replicas = &newReplicas
			assert.NoError(t, cl.Update(context.TODO(), logger, created))
			assert.NoError(t, cl.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			require.NotNil(t, retrieved.Spec.Replicas)
			assert.EqualValues(t, 10, *retrieved.Spec.Replicas)
			assert.EqualValues(t, 2, retrieved.Generation) // Generation updated
		})

		t.Run("update object with same spec", func(t *testing.T) {
			created, retrieved := createAndGetDeployment(t, logger, cl)
			assert.NoError(t, cl.Update(context.TODO(), logger, created))
			assert.NoError(t, cl.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			require.NotNil(t, retrieved.Spec.Replicas)
			assert.EqualValues(t, 1, *retrieved.Spec.Replicas)
			assert.EqualValues(t, 1, retrieved.Generation) // Generation updated
		})

		t.Run("status update", func(t *testing.T) {
			created, retrieved := createAndGetSecret(t, logger, cl)
			assert.NoError(t, cl.Status().Update(context.TODO(), logger, created))
			assert.NoError(t, cl.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			assert.EqualValues(t, 1, retrieved.Generation) // Generation not changed
		})

		t.Run("patch", func(t *testing.T) {
			created, retrieved := createAndGetSecret(t, logger, cl)
			annotations := make(map[string]string)
			annotations["foo"] = "bar"

			mergePatch, err := json.Marshal(map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": annotations,
				},
			})
			require.NoError(t, err)
			assert.NoError(t, cl.Patch(context.TODO(), logger, created, client.RawPatch(types.MergePatchType, mergePatch)))
			assert.NoError(t, cl.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			assert.Equal(t, annotations, retrieved.GetObjectMeta().GetAnnotations())
		})

		t.Run("status patch", func(t *testing.T) {
			_, retrieved := createAndGetDeployment(t, logger, cl)
			depPatch := client.MergeFrom(retrieved.DeepCopy())
			retrieved.Status.Replicas = 1
			assert.NoError(t, cl.Status().Patch(context.TODO(), logger, retrieved, depPatch))
		})

		t.Run("delete", func(t *testing.T) {
			created, retrieved := createAndGetSecret(t, logger, cl)
			assert.NoError(t, cl.Delete(context.TODO(), logger, created))
			err := cl.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved)
			require.Error(t, err)
			assert.True(t, errs.IsNotFound(err))
		})

		// t.Run("deleteAllOf", func(t *testing.T) {
		// 	created, retrieved := createAndGetDeployment(t,logger, client)
		// 	dep2 := retrieved.DeepCopy()
		// 	dep2.Name = dep2.Name + "-2"
		// 	dep2.ResourceVersion = ""
		// 	assert.NoError(t, fclient.Create(context.TODO(), logger, dep2))

		// 	assert.NoError(t, fclient.DeleteAllOf(context.TODO(), retrieved, client.InNamespace("somenamespace"), client.MatchingLabels(retrieved.ObjectMeta.Labels)))
		// 	err := fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved)
		// 	require.Error(t, err)
		// 	assert.True(t, errs.IsNotFound(err))

		// 	err = fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: dep2.Name}, dep2)
		// 	require.Error(t, err)
		// 	assert.True(t, errs.IsNotFound(err))
		// })
	})

	t.Run("mock methods OK", func(t *testing.T) {
		expectedErr := errors.New("oopsie woopsie")
		created, _ := createAndGetSecret(t, logger, cl)

		t.Run("mock Get", func(t *testing.T) {
			defer func() { cl.MockGet = nil }()
			cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return expectedErr
			}
			assert.EqualError(t, cl.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, &v1.Secret{}), expectedErr.Error())
		})

		t.Run("mock List", func(t *testing.T) {
			defer func() { cl.MockList = nil }()
			cl.MockList = func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return expectedErr
			}
			assert.EqualError(t, cl.List(context.TODO(), &v1.SecretList{}, client.InNamespace("somenamespace")), expectedErr.Error())
		})

		t.Run("mock Create", func(t *testing.T) {
			defer func() { cl.MockCreate = nil }()
			cl.MockCreate = func(ctx context.Context, logger logr.Logger, obj client.Object, option ...client.CreateOption) error {
				return expectedErr
			}
			assert.EqualError(t, cl.Create(context.TODO(), logger, &v1.Secret{}), expectedErr.Error())
		})

		t.Run("mock Update", func(t *testing.T) {
			defer func() { cl.MockUpdate = nil }()
			cl.MockUpdate = func(ctx context.Context, logger logr.Logger, obj client.Object, option ...client.UpdateOption) error {
				return expectedErr
			}
			assert.EqualError(t, cl.Update(context.TODO(), logger, &v1.Secret{}), expectedErr.Error())
		})

		t.Run("mock Patch", func(t *testing.T) {
			defer func() { cl.MockPatch = nil }()
			cl.MockPatch = func(ctx context.Context, logger logr.Logger, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				return expectedErr
			}
			assert.EqualError(t, cl.Patch(context.TODO(), logger, &v1.Secret{}, client.RawPatch(types.MergePatchType, []byte{})), expectedErr.Error())
		})

		t.Run("mock Delete", func(t *testing.T) {
			defer func() { cl.MockDelete = nil }()
			cl.MockDelete = func(ctx context.Context, logger logr.Logger, obj client.Object, opts ...client.DeleteOption) error {
				return expectedErr
			}
			assert.EqualError(t, cl.Delete(context.TODO(), logger, &v1.Secret{}), expectedErr.Error())
		})

		// t.Run("mock DeleteAllOf", func(t *testing.T) {
		// 	defer func() { fclient.MockDeleteAllOf = nil }()
		// 	fclient.MockDeleteAllOf = func(ctx context.Context, logger logr.Logger, obj client.Object, opts ...client.DeleteAllOfOption) error {
		// 		return expectedErr
		// 	}
		// 	assert.EqualError(t, fclient.DeleteAllOf(context.TODO(), &v1.Secret{}), expectedErr.Error())
		// })

		t.Run("mock Status Update", func(t *testing.T) {
			defer func() { cl.MockStatusUpdate = nil }()
			cl.MockStatusUpdate = func(ctx context.Context, logger logr.Logger, obj client.Object, opts ...client.UpdateOption) error {
				return expectedErr
			}
			assert.EqualError(t, cl.MockStatusUpdate(context.TODO(), logger, &v1.Secret{}), expectedErr.Error())
		})

		t.Run("mock Status Patch", func(t *testing.T) {
			defer func() { cl.MockStatusPatch = nil }()
			cl.MockStatusPatch = func(ctx context.Context, logger logr.Logger, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				return expectedErr
			}
			assert.EqualError(t, cl.MockStatusPatch(context.TODO(), logger, &v1.Secret{}, client.RawPatch(types.MergePatchType, []byte{})), expectedErr.Error())
		})
	})
}

func createAndGetSecret(t *testing.T, logger logr.Logger, fclient *FakeClient) (*v1.Secret, *v1.Secret) {
	key := types.NamespacedName{Namespace: "somenamespace", Name: "somename" + uuid.Must(uuid.NewV4()).String()}
	data := make(map[string]string)
	data["key"] = "value"
	created := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		StringData: data,
	}

	// Create
	assert.NoError(t, fclient.Create(context.TODO(), logger, created))

	// Get
	secret := &v1.Secret{}
	assert.NoError(t, fclient.Get(context.TODO(), key, secret))
	assert.Equal(t, created, secret)
	assert.EqualValues(t, 1, secret.Generation)

	return created, secret
}

func createAndGetDeployment(t *testing.T, logger logr.Logger, fclient *FakeClient) (*appsv1.Deployment, *appsv1.Deployment) {
	key := types.NamespacedName{Namespace: "somenamespace", Name: "somename" + uuid.Must(uuid.NewV4()).String()}
	replicas := int32(1)
	created := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	// Create
	assert.NoError(t, fclient.Create(context.TODO(), logger, created))

	// Get
	dep := &appsv1.Deployment{}
	assert.NoError(t, fclient.Get(context.TODO(), key, dep))
	assert.Equal(t, created, dep)
	assert.EqualValues(t, 1, dep.Generation)

	return created, dep
}
