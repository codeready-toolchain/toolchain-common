package client_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/codeready-toolchain/api/pkg/apis"
	applyCl "github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	. "github.com/codeready-toolchain/toolchain-common/pkg/test"

	authv1 "github.com/openshift/api/authorization/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestApplySingle(t *testing.T) {
	// given
	s := addToScheme(t)

	defaultService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registration-service",
			Namespace: "toolchain-host-operator",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"run": "registration-service",
			},
		},
	}

	modifiedService := defaultService.DeepCopyObject().(*corev1.Service)
	modifiedService.Spec.Selector["run"] = "all-services"

	defaultCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registration-service",
			Namespace: "toolchain-host-operator",
		},
		Data: map[string]string{
			"first-param": "first-value",
		},
	}

	modifiedCm := defaultCm.DeepCopyObject().(*corev1.ConfigMap)
	modifiedCm.Data["first-param"] = "second-value"

	t.Run("updates of service object", func(t *testing.T) {
		// given
		namespacedName := types.NamespacedName{Namespace: "toolchain-host-operator", Name: "registration-service"}

		t.Run("when using forceUpdate=true, it should not update when specs are same", func(t *testing.T) {
			// given
			cl, _ := newClient(t, s)
			obj := defaultService.DeepCopy()
			_, err := cl.CreateOrUpdateObject(obj, true, nil)
			require.NoError(t, err)
			originalGeneration := obj.GetGeneration()

			// when updating with the same obj again
			createdOrChanged, err := cl.CreateOrUpdateObject(obj, true, nil)

			// then
			require.NoError(t, err)
			assert.False(t, createdOrChanged) // resource was not updated on the server, so returned value is `false`
			updateGeneration := obj.GetGeneration()
			assert.Equal(t, originalGeneration, updateGeneration)
		})

		t.Run("when using forceUpdate=true, it should update when specs are different", func(t *testing.T) {
			// given
			cl, _ := newClient(t, s)
			obj := defaultService.DeepCopy()
			_, err := cl.CreateOrUpdateObject(obj, true, nil)
			require.NoError(t, err)
			originalGeneration := obj.GetGeneration()

			// when updating with the modified obj
			modifiedObj := modifiedService.DeepCopy()
			modifiedObj.ObjectMeta.Generation = obj.GetGeneration()
			createdOrChanged, err := cl.CreateOrUpdateObject(modifiedObj, true, nil)

			// then
			require.NoError(t, err)
			assert.True(t, createdOrChanged) // resource was updated on the server, so returned value if `true`
			updateGeneration := modifiedObj.GetGeneration()
			assert.Equal(t, originalGeneration+1, updateGeneration)
		})

		t.Run("when using forceUpdate=false, it should update when spec is different", func(t *testing.T) {
			// given
			cl, cli := newClient(t, s)
			_, err := cl.CreateOrUpdateObject(defaultService.DeepCopyObject(), true, nil)
			require.NoError(t, err)

			// when
			createdOrChanged, err := cl.CreateOrUpdateObject(modifiedService.DeepCopyObject(), false, nil)

			// then
			require.NoError(t, err)
			assert.True(t, createdOrChanged)
			service := &corev1.Service{}
			err = cli.Get(context.TODO(), namespacedName, service)
			require.NoError(t, err)
			assert.Equal(t, "all-services", service.Spec.Selector["run"])
		})

		t.Run("when using forceUpdate=false, it should NOT update when using same object", func(t *testing.T) {
			// given
			cl, _ := newClient(t, s)
			_, err := cl.CreateOrUpdateObject(defaultService.DeepCopyObject(), true, nil)
			require.NoError(t, err)

			// when
			createdOrChanged, err := cl.CreateOrUpdateObject(defaultService.DeepCopyObject(), false, nil)

			// then
			require.NoError(t, err)
			assert.False(t, createdOrChanged)
		})

		t.Run("when object is missing, it should create it no matter what is set as forceUpdate", func(t *testing.T) {
			// given
			cl, cli := newClient(t, s)
			deployment := &v1.Deployment{}

			// when
			createdOrChanged, err := cl.CreateOrUpdateObject(modifiedService.DeepCopyObject(), false, deployment)

			// then
			require.NoError(t, err)
			assert.True(t, createdOrChanged)
			service := &corev1.Service{}
			err = cli.Get(context.TODO(), namespacedName, service)
			require.NoError(t, err)
			assert.Equal(t, "all-services", service.Spec.Selector["run"])
			assert.NotEmpty(t, service.OwnerReferences)
		})

		t.Run("when object cannot be retrieved because of any error, then it should fail", func(t *testing.T) {
			// given
			cl, cli := newClient(t, s)
			cli.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
				return fmt.Errorf("unable to get")
			}

			// when
			createdOrChanged, err := cl.CreateOrUpdateObject(modifiedService.DeepCopyObject(), false, nil)

			// then
			require.Error(t, err)
			assert.False(t, createdOrChanged)
			assert.Contains(t, err.Error(), "unable to get the resource")
		})
	})

	t.Run("when using forceUpdate=false, it should update ConfigMap when data field is different", func(t *testing.T) {
		// given
		cl, cli := newClient(t, s)
		_, err := cl.CreateOrUpdateObject(defaultCm.DeepCopyObject(), true, nil)
		require.NoError(t, err)

		// when
		createdOrChanged, err := cl.CreateOrUpdateObject(modifiedCm.DeepCopyObject(), false, nil)

		// then
		require.NoError(t, err)
		assert.True(t, createdOrChanged)
		configMap := &corev1.ConfigMap{}
		namespacedName := types.NamespacedName{Namespace: "toolchain-host-operator", Name: "registration-service"}
		err = cli.Get(context.TODO(), namespacedName, configMap)
		require.NoError(t, err)
		assert.Equal(t, "second-value", configMap.Data["first-param"])
	})
}

func newClient(t *testing.T, s *runtime.Scheme) (*applyCl.ApplyClient, *test.FakeClient) {
	cli := NewFakeClient(t)
	cli.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
		// force the Generation to `1` for newly created objects
		m, err := meta.Accessor(obj)
		if err != nil {
			return err
		}
		m.SetGeneration(1)
		return cli.Client.Create(ctx, obj, opts...)
	}
	cli.MockUpdate = func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
		// compare the specs (only) and only increment the generation if something changed
		// (the server will check the object metadata, but we're skipping this here)
		if svc, ok := obj.(*corev1.Service); ok {
			existing := corev1.Service{}
			if err := cli.Get(ctx, types.NamespacedName{Namespace: svc.GetNamespace(), Name: svc.GetName()}, &existing); err != nil {
				return err
			}
			if !reflect.DeepEqual(existing.Spec, svc.Spec) { // Service has a `spec` field
				svc.SetGeneration(existing.GetGeneration() + 1)
			}
		} else if cm, ok := obj.(*corev1.ConfigMap); ok {
			existing := corev1.ConfigMap{}
			if err := cli.Get(ctx, types.NamespacedName{Namespace: cm.GetNamespace(), Name: cm.GetName()}, &existing); err != nil {
				return err
			}
			if !reflect.DeepEqual(existing.Data, cm.Data) { // ConfigMap has a `data` field
				cm.SetGeneration(existing.GetGeneration() + 1)
			}
		}
		return cli.Client.Update(ctx, obj)
	}
	return applyCl.NewApplyClient(cli, s), cli
}

func addToScheme(t *testing.T) *runtime.Scheme {
	s := scheme.Scheme
	err := authv1.Install(s)
	require.NoError(t, err)
	err = apis.AddToScheme(s)
	require.NoError(t, err)
	return s
}
