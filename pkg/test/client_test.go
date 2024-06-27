package test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
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

	t.Run("default methods OK", func(t *testing.T) {
		t.Run("list", func(t *testing.T) {
			created, _ := createAndGetSecret(t, fclient)
			secretList := &v1.SecretList{}
			require.NoError(t, fclient.List(context.TODO(), secretList, client.InNamespace("somenamespace")))
			require.Len(t, secretList.Items, 1)
			assert.Equal(t, *created, secretList.Items[0])
		})

		t.Run("update object with stringData", func(t *testing.T) {
			created, retrieved := createAndGetSecret(t, fclient)
			created.StringData["key"] = "updated"
			require.NoError(t, fclient.Update(context.TODO(), created))
			require.NoError(t, fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			assert.Equal(t, "updated", retrieved.StringData["key"])
			assert.EqualValues(t, 2, retrieved.Generation) // Generation updated
		})

		t.Run("update object with the same stringData", func(t *testing.T) {
			created, retrieved := createAndGetSecret(t, fclient)
			require.NoError(t, fclient.Update(context.TODO(), created))
			require.NoError(t, fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			assert.Equal(t, "value", retrieved.StringData["key"])
			assert.EqualValues(t, 1, retrieved.Generation) // Generation not updated
		})

		t.Run("update object with data", func(t *testing.T) {
			key := types.NamespacedName{Namespace: "somenamespace", Name: "somename" + uuid.NewString()}
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
			require.NoError(t, fclient.Create(context.TODO(), created))
			// Get
			secret := &v1.Secret{}
			require.NoError(t, fclient.Get(context.TODO(), key, secret))
			assert.Equal(t, created, secret)
			assert.EqualValues(t, 1, secret.Generation)

			data["newkey"] = []byte("newvalue")
			created.Data = data
			require.NoError(t, fclient.Update(context.TODO(), created))
			require.NoError(t, fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, secret))
			assert.Equal(t, []byte("value"), secret.Data["key"])
			assert.Equal(t, []byte("newvalue"), secret.Data["newkey"])
			assert.EqualValues(t, 2, secret.Generation) // Generation updated
		})

		t.Run("update object with spec", func(t *testing.T) {
			created, retrieved := createAndGetDeployment(t, fclient)
			newReplicas := int32(10)
			created.Spec.Replicas = &newReplicas
			require.NoError(t, fclient.Update(context.TODO(), created))
			require.NoError(t, fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			require.NotNil(t, retrieved.Spec.Replicas)
			assert.EqualValues(t, 10, *retrieved.Spec.Replicas)
			assert.EqualValues(t, 2, retrieved.Generation) // Generation updated
		})

		t.Run("update object with same spec", func(t *testing.T) {
			created, retrieved := createAndGetDeployment(t, fclient)
			require.NoError(t, fclient.Update(context.TODO(), created))
			require.NoError(t, fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			require.NotNil(t, retrieved.Spec.Replicas)
			assert.EqualValues(t, 1, *retrieved.Spec.Replicas)
			assert.EqualValues(t, 1, retrieved.Generation) // Generation updated
		})

		t.Run("no error in blank status update", func(t *testing.T) {
			created, retrieved := createAndGetDeployment(t, fclient)
			require.NoError(t, fclient.Status().Update(context.TODO(), created))
			require.NoError(t, fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			assert.EqualValues(t, 1, retrieved.Generation) // Generation not changed, since blank update
		})

		t.Run("status update with actual update in status", func(t *testing.T) {
			created, retrieved := createAndGetDeployment(t, fclient)
			created.Status.Replicas = 2
			require.NoError(t, fclient.Status().Update(context.TODO(), created))
			require.NoError(t, fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			assert.EqualValues(t, 2, retrieved.Status.Replicas) // replicas count changed to 2
		})

		t.Run("status update fails when the objects being updated doesn't have status", func(t *testing.T) {
			created, _ := createAndGetSecret(t, fclient)
			require.Error(t, fclient.Status().Update(context.TODO(), created))
		})

		t.Run("patch", func(t *testing.T) {
			created, retrieved := createAndGetSecret(t, fclient)
			annotations := make(map[string]string)
			annotations["foo"] = "bar"

			mergePatch, err := json.Marshal(map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": annotations,
				},
			})
			require.NoError(t, err)
			require.NoError(t, fclient.Patch(context.TODO(), created, client.RawPatch(types.MergePatchType, mergePatch)))
			require.NoError(t, fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved))
			assert.Equal(t, annotations, retrieved.GetObjectMeta().GetAnnotations())
		})

		t.Run("status patch", func(t *testing.T) {
			_, retrieved := createAndGetDeployment(t, fclient)
			depPatch := client.MergeFrom(retrieved.DeepCopy())
			retrieved.Status.Replicas = 1
			require.NoError(t, fclient.Status().Patch(context.TODO(), retrieved, depPatch))
		})

		t.Run("delete", func(t *testing.T) {
			created, retrieved := createAndGetSecret(t, fclient)
			require.NoError(t, fclient.Delete(context.TODO(), created))
			err := fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved)
			require.Error(t, err)
			assert.True(t, errs.IsNotFound(err))
		})

		t.Run("deleteAllOf", func(t *testing.T) {
			created, retrieved := createAndGetDeployment(t, fclient)
			dep2 := retrieved.DeepCopy()
			dep2.Name = dep2.Name + "-2"
			dep2.ResourceVersion = ""
			require.NoError(t, fclient.Create(context.TODO(), dep2))

			require.NoError(t, fclient.DeleteAllOf(context.TODO(), retrieved, client.InNamespace("somenamespace"), client.MatchingLabels(retrieved.ObjectMeta.Labels)))
			err := fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, retrieved)
			require.Error(t, err)
			assert.True(t, errs.IsNotFound(err))

			err = fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: dep2.Name}, dep2)
			require.Error(t, err)
			assert.True(t, errs.IsNotFound(err))
		})
	})

	t.Run("mock methods OK", func(t *testing.T) {
		expectedErr := errors.New("oopsie woopsie")
		created, _ := createAndGetSecret(t, fclient)

		t.Run("mock Get", func(t *testing.T) {
			defer func() { fclient.MockGet = nil }()
			fclient.MockGet = func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return expectedErr
			}
			require.EqualError(t, fclient.Get(context.TODO(), types.NamespacedName{Namespace: "somenamespace", Name: created.Name}, &v1.Secret{}), expectedErr.Error())
		})

		t.Run("mock List", func(t *testing.T) {
			defer func() { fclient.MockList = nil }()
			fclient.MockList = func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return expectedErr
			}
			require.EqualError(t, fclient.List(context.TODO(), &v1.SecretList{}, client.InNamespace("somenamespace")), expectedErr.Error())
		})

		t.Run("mock Create", func(t *testing.T) {
			defer func() { fclient.MockCreate = nil }()
			fclient.MockCreate = func(ctx context.Context, obj client.Object, option ...client.CreateOption) error {
				return expectedErr
			}
			require.EqualError(t, fclient.Create(context.TODO(), &v1.Secret{}), expectedErr.Error())
		})

		t.Run("mock Update", func(t *testing.T) {
			defer func() { fclient.MockUpdate = nil }()
			fclient.MockUpdate = func(ctx context.Context, obj client.Object, option ...client.UpdateOption) error {
				return expectedErr
			}
			require.EqualError(t, fclient.Update(context.TODO(), &v1.Secret{}), expectedErr.Error())
		})

		t.Run("mock Patch", func(t *testing.T) {
			defer func() { fclient.MockPatch = nil }()
			fclient.MockPatch = func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				return expectedErr
			}
			require.EqualError(t, fclient.Patch(context.TODO(), &v1.Secret{}, client.RawPatch(types.MergePatchType, []byte{})), expectedErr.Error())
		})

		t.Run("mock Delete", func(t *testing.T) {
			defer func() { fclient.MockDelete = nil }()
			fclient.MockDelete = func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
				return expectedErr
			}
			require.EqualError(t, fclient.Delete(context.TODO(), &v1.Secret{}), expectedErr.Error())
		})

		t.Run("mock DeleteAllOf", func(t *testing.T) {
			defer func() { fclient.MockDeleteAllOf = nil }()
			fclient.MockDeleteAllOf = func(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
				return expectedErr
			}
			require.EqualError(t, fclient.DeleteAllOf(context.TODO(), &v1.Secret{}), expectedErr.Error())
		})

		t.Run("mock Status Update", func(t *testing.T) {
			defer func() { fclient.MockStatusUpdate = nil }()
			fclient.MockStatusUpdate = func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				return expectedErr
			}
			require.EqualError(t, fclient.MockStatusUpdate(context.TODO(), &v1.Secret{}), expectedErr.Error())
		})

		t.Run("mock Status Patch", func(t *testing.T) {
			defer func() { fclient.MockStatusPatch = nil }()
			fclient.MockStatusPatch = func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
				return expectedErr
			}
			require.EqualError(t, fclient.MockStatusPatch(context.TODO(), &v1.Secret{}, client.RawPatch(types.MergePatchType, []byte{})), expectedErr.Error())
		})
	})
}

func createAndGetSecret(t *testing.T, fclient *FakeClient) (*v1.Secret, *v1.Secret) {
	key := types.NamespacedName{Namespace: "somenamespace", Name: "somename" + uuid.NewString()}
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
	require.NoError(t, fclient.Create(context.TODO(), created))

	// Get
	secret := &v1.Secret{}
	require.NoError(t, fclient.Get(context.TODO(), key, secret))
	assert.Equal(t, created, secret)
	assert.EqualValues(t, 1, secret.Generation)

	return created, secret
}

func createAndGetDeployment(t *testing.T, fclient *FakeClient) (*appsv1.Deployment, *appsv1.Deployment) {
	key := types.NamespacedName{Namespace: "somenamespace", Name: "somename" + uuid.NewString()}
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
	require.NoError(t, fclient.Create(context.TODO(), created))

	// Get
	dep := &appsv1.Deployment{}
	require.NoError(t, fclient.Get(context.TODO(), key, dep))
	assert.Equal(t, created, dep)
	assert.EqualValues(t, 1, dep.Generation)

	return created, dep
}
