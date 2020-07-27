package controller

import (
	"os"
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLoadFromConfigMap(t *testing.T) {
	restore := test.SetEnvVarAndRestore(t, "WATCH_NAMESPACE", "toolchain-member-operator")
	defer restore()
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
				"test-test": "test-test",
			},
		}

		cl := test.NewFakeClient(t, configMap)

		// when
		err := LoadFromConfigMap("MEMBER_OPERATOR", "MEMBER_OPERATOR_CONFIG_MAP_NAME", cl)

		// then
		require.NoError(t, err)

		// test env vars are parsed and created correctly
		testTest := os.Getenv("MEMBER_OPERATOR_TEST_TEST")
		assert.Equal(t, testTest, "test-test")
	})

	t.Run("configMap not found", func(t *testing.T) {
		// given
		restore := test.SetEnvVarAndRestore(t, "MEMBER_OPERATOR_CONFIG_MAP_NAME", "test-config")
		defer restore()

		cl := test.NewFakeClient(t)

		// when
		err := LoadFromConfigMap("HOST_OPERATOR", "MEMBER_OPERATOR_CONFIG_MAP_NAME", cl)

		// then
		require.NoError(t, err)
	})
}

func TestLoadFromSecret(t *testing.T) {
	restore := test.SetEnvVarAndRestore(t, "WATCH_NAMESPACE", "toolchain-host-operator")
	defer restore()
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
				"test.test": []byte("test-test"),
			},
		}

		cl := test.NewFakeClient(t, secret)

		// when
		err := LoadFromSecret("HOST_OPERATOR", "HOST_OPERATOR_SECRET_NAME", cl)

		// then
		require.NoError(t, err)

		// test env vars are parsed and created correctly
		testTest := os.Getenv("HOST_OPERATOR_TEST_TEST")
		assert.Equal(t, testTest, "test-test")
	})

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
}
