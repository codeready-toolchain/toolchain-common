package configuration

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		data := map[string]string{
			"super-special-key": "super-special-value",
		}
		cl := test.NewFakeClient(t, createConfigMap("test-config", "toolchain-host-operator", data))

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

		data := map[string]string{
			"test-key-one": "test-value-one",
		}
		cl := test.NewFakeClient(t, createConfigMap("test-config", "toolchain-host-operator", data))

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
		restore := test.SetEnvVarsAndRestore(t,
			test.Env("MEMBER_OPERATOR_CONFIG_MAP_NAME", "test-config"),
			test.Env("MEMBER_OPERATOR_TEST_KEY", ""))
		defer restore()

		data := map[string]string{
			"test-key": "test-value",
		}
		cl := test.NewFakeClient(t, createConfigMap("test-config", "toolchain-member-operator", data))

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
		data := map[string][]byte{
			"special.key": []byte("special-value"),
		}
		cl := test.NewFakeClient(t, createSecret("test-secret", "toolchain-host-operator", data))

		// when
		err := LoadFromSecret("HOST_OPERATOR", "HOST_OPERATOR_SECRET_NAME", cl)

		// then
		require.NoError(t, err)

		// test that the secret was not found since no secret name was set
		testTest := os.Getenv("HOST_OPERATOR_SPECIAL_KEY")
		assert.Equal(t, "", testTest)
	})
	t.Run("cannot get secret", func(t *testing.T) {
		// given
		restore := test.SetEnvVarAndRestore(t, "HOST_OPERATOR_SECRET_NAME", "test-secret")
		defer restore()

		data := map[string][]byte{
			"test.key.secret": []byte("test-value-secret"),
		}
		cl := test.NewFakeClient(t, createSecret("test-secret", "toolchain-host-operator", data))

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
		restore := test.SetEnvVarsAndRestore(t,
			test.Env("HOST_OPERATOR_SECRET_NAME", "test-secret"),
			test.Env("HOST_OPERATOR_TEST_KEY", ""))
		defer restore()

		data := map[string][]byte{
			"test.key": []byte("test-value"),
		}
		cl := test.NewFakeClient(t, createSecret("test-secret", "toolchain-host-operator", data))

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

		data := map[string][]byte{
			"test.key": []byte("test-value"),
		}
		cl := test.NewFakeClient(t, createSecret("test-secret", "toolchain-host-operator", data))

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

		data := map[string]string{
			"test-key": "test-value",
		}
		cl := test.NewFakeClient(t, createConfigMap("test-config", "toolchain-host-operator", data))

		// when
		err := LoadFromSecret("HOST_OPERATOR", "HOST_OPERATOR_CONFIG_MAP_NAME", cl)

		// then
		require.Error(t, err)
		assert.Equal(t, "WATCH_NAMESPACE must be set", err.Error())
	})
}

func createSecret(name, namespace string, data map[string][]byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func createConfigMap(name, namespace string, data map[string]string) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}
