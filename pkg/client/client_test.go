package client_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/codeready-toolchain/api/pkg/apis"
	applyCl "github.com/codeready-toolchain/toolchain-common/pkg/client"
	. "github.com/codeready-toolchain/toolchain-common/pkg/test"

	authv1 "github.com/openshift/api/authorization/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

		t.Run("when using forceUpdate=true, it should update even when spec is same", func(t *testing.T) {
			// given
			cli := NewFakeClient(t)
			cl := applyCl.NewApplyClient(cli, s)
			_, err := cl.CreateOrUpdateObject(defaultService.DeepCopyObject(), true, nil)
			require.NoError(t, err)

			// when
			createdOrChanged, err := cl.CreateOrUpdateObject(modifiedService.DeepCopyObject(), true, nil)

			// then
			require.NoError(t, err)
			assert.True(t, createdOrChanged)
		})

		t.Run("when using forceUpdate=false, it should update when spec is different", func(t *testing.T) {
			// given
			cli := NewFakeClient(t)
			cl := applyCl.NewApplyClient(cli, s)
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
			cli := NewFakeClient(t)
			cl := applyCl.NewApplyClient(cli, s)
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
			cli := NewFakeClient(t)
			cl := applyCl.NewApplyClient(cli, s)
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
			cli := NewFakeClient(t)
			cl := applyCl.NewApplyClient(cli, s)
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
		cli := NewFakeClient(t)
		cl := applyCl.NewApplyClient(cli, s)
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

func addToScheme(t *testing.T) *runtime.Scheme {
	s := scheme.Scheme
	err := authv1.Install(s)
	require.NoError(t, err)
	err = apis.AddToScheme(s)
	require.NoError(t, err)
	return s
}
