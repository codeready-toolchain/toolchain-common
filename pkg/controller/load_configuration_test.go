package controller

import (
	"context"
	"errors"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLoadFromConfigMap(t *testing.T) {
	restore := test.SetEnvVarAndRestore(t, "WATCH_NAMESPACE", "toolchain-member-operator")
	defer restore()

	t.Run("configMap not found", func(t *testing.T) {
		// given
		restore := test.SetEnvVarAndRestore(t, "MEMBER_OPERATOR_CONFIG_MAP_NAME", "test-config")
		defer restore()

		cl := test.NewFakeClient(t)

		// when
		err := LoadFromConfigMap("MEMBER_OPERATOR", "MEMBER_OPERATOR_CONFIG_MAP_NAME", cl)

		// then
		require.NoError(t, err)
	})
	t.Run("no config name set", func(t *testing.T) {
		// given
		configMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config",
				Namespace: "toolchain-member-operator",
			},
			Data: map[string]string{
				"super-special-key": "super-special-value",
			},
		}

		cl := test.NewFakeClient(t, configMap)

		// when
		err := LoadFromSecret("HOST_OPERATOR", "HOST_OPERATOR_CONFIG_MAP_NAME", cl)

		// then
		require.NoError(t, err)

		// test that the secret was not found since no secret name was set
		testTest := os.Getenv("HOST_OPERATOR_SUPER_SPECIAL_KEY")
		assert.Equal(t, "", testTest)
	})
	t.Run("cannot get configmap", func(t *testing.T) {
		// given
		restore := test.SetEnvVarAndRestore(t, "MEMBER_OPERATOR_CONFIG_MAP_NAME", "test-config")
		defer restore()

		configMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config",
				Namespace: "toolchain-member-operator",
			},
			Data: map[string]string{
				"test-key-one": "test-value-one",
			},
		}

		cl := test.NewFakeClient(t, configMap)

		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
			return errors.New("oopsie woopsie")
		}

		// when
		err := LoadFromConfigMap("MEMBER_OPERATOR", "MEMBER_OPERATOR_CONFIG_MAP_NAME", cl)

		// then
		require.Error(t, err)
		assert.Equal(t, "oopsie woopsie", err.Error())

		// test env vars are parsed and created correctly
		testTest := os.Getenv("MEMBER_OPERATOR_TEST_KEY_ONE")
		assert.Equal(t, testTest, "")
	})
	t.Run("env overwrite", func(t *testing.T) {
		// given
		restore := test.SetEnvVarAndRestore(t, "MEMBER_OPERATOR_CONFIG_MAP_NAME", "test-config")
		defer restore()

		configMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config",
				Namespace: "toolchain-member-operator",
			},
			Data: map[string]string{
				"test-key": "test-value",
			},
		}

		cl := test.NewFakeClient(t, configMap)

		// when
		err := LoadFromConfigMap("MEMBER_OPERATOR", "MEMBER_OPERATOR_CONFIG_MAP_NAME", cl)

		// then
		require.NoError(t, err)

		// test env vars are parsed and created correctly
		testTest := os.Getenv("MEMBER_OPERATOR_TEST_KEY")
		assert.Equal(t, testTest, "test-value")
	})
}

func TestLoadFromSecret(t *testing.T) {
	restore := test.SetEnvVarAndRestore(t, "WATCH_NAMESPACE", "toolchain-host-operator")
	defer restore()
	t.Run("secret not found", func(t *testing.T) {
		// given
		restore := test.SetEnvVarAndRestore(t, "HOST_OPERATOR_SECRET_NAME", "test-secret")
		defer restore()

		cl := test.NewFakeClient(t)

		// when
		err := LoadFromConfigMap("HOST_OPERATOR", "HOST_OPERATOR_SECRET_NAME", cl)

		// then
		require.NoError(t, err)
	})
	t.Run("no secret name set", func(t *testing.T) {
		// given
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "toolchain-host-operator",
			},
			Data: map[string][]byte{
				"special.key": []byte("special-value"),
			},
		}

		cl := test.NewFakeClient(t, secret)

		// when
		err := LoadFromSecret("HOST_OPERATOR", "HOST_OPERATOR_SECRET_NAME", cl)

		// then
		require.NoError(t, err)

		// test that the secret was not found since no secret name was set
		testTest := os.Getenv("HOST_OPERATOR_SPECIAL_KEY")
		assert.Equal(t, "", testTest)
	})
	t.Run("env overwrite", func(t *testing.T) {
		// given
		restore := test.SetEnvVarAndRestore(t, "HOST_OPERATOR_SECRET_NAME", "test-secret")
		defer restore()

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "toolchain-host-operator",
			},
			Data: map[string][]byte{
				"test.key.secret": []byte("test-value-secret"),
			},
		}

		cl := test.NewFakeClient(t, secret)

		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
			return errors.New("oopsie woopsie")
		}

		// when
		err := LoadFromConfigMap("HOST_OPERATOR", "HOST_OPERATOR_SECRET_NAME", cl)

		// then
		require.Error(t, err)
		assert.Equal(t, "oopsie woopsie", err.Error())

		// test env vars are parsed and created correctly
		testTest := os.Getenv("HOST_OPERATOR_TEST_KEY_SECRET")
		assert.Equal(t, testTest, "")
	})
	t.Run("env overwrite", func(t *testing.T) {
		// given
		restore := test.SetEnvVarAndRestore(t, "HOST_OPERATOR_SECRET_NAME", "test-secret")
		defer restore()

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "toolchain-host-operator",
			},
			Data: map[string][]byte{
				"test.key": []byte("test-value"),
			},
		}

		cl := test.NewFakeClient(t, secret)

		// when
		err := LoadFromSecret("HOST_OPERATOR", "HOST_OPERATOR_SECRET_NAME", cl)

		// then
		require.NoError(t, err)

		// test env vars are parsed and created correctly
		testTest := os.Getenv("HOST_OPERATOR_TEST_KEY")
		assert.Equal(t, testTest, "test-value")
	})
}

func TestNoWatchNamespaceSetWhenLoadingSecret(t *testing.T) {
	t.Run("no watch namespace", func(t *testing.T) {
		// given
		restore := test.SetEnvVarAndRestore(t, "HOST_OPERATOR_SECRET_NAME", "test-secret")
		defer restore()

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "toolchain-host-operator",
			},
			Data: map[string][]byte{
				"test.key": []byte("test-value"),
			},
		}

		cl := test.NewFakeClient(t, secret)

		// when
		err := LoadFromSecret("HOST_OPERATOR", "HOST_OPERATOR_SECRET_NAME", cl)

		// then
		require.Error(t, err)
		assert.Equal(t, "WATCH_NAMESPACE must be set", err.Error())
	})
}

func TestNoWatchNamespaceSetWhenLoadingConfigMap(t *testing.T) {
	t.Run("no watch namespace", func(t *testing.T) {
		// given
		restore := test.SetEnvVarAndRestore(t, "HOST_OPERATOR_CONFIG_MAP_NAME", "test-config")
		defer restore()

		configMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config",
				Namespace: "toolchain-host-operator",
			},
			Data: map[string]string{
				"test-key": "test-value",
			},
		}

		cl := test.NewFakeClient(t, configMap)

		// when
		err := LoadFromSecret("HOST_OPERATOR", "HOST_OPERATOR_CONFIG_MAP_NAME", cl)

		// then
		require.Error(t, err)
		assert.Equal(t, "WATCH_NAMESPACE must be set", err.Error())
	})
}
