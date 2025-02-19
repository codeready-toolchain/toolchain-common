package configuration

import (
	"context"
	"fmt"
	"sync"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	testconfig "github.com/codeready-toolchain/toolchain-common/pkg/test/config"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

func TestCache(t *testing.T) {
	// given
	cl := test.NewFakeClient(t)

	t.Run("WATCH_NAMESPACE not set", func(t *testing.T) {
		// when
		actual, secrets, err := GetConfig(cl, &toolchainv1alpha1.ToolchainConfig{})

		// then
		require.EqualError(t, err, "failed to get watch namespace: WATCH_NAMESPACE must be set")
		require.Nil(t, actual)
		require.Empty(t, secrets)
	})

	restore := test.SetEnvVarAndRestore(t, "WATCH_NAMESPACE", test.HostOperatorNs)
	defer restore()
	t.Run("empty cache", func(t *testing.T) {
		// when
		actual, secrets, err := GetConfig(cl, &toolchainv1alpha1.ToolchainConfig{})

		// then
		require.NoError(t, err)
		require.Nil(t, actual)
		require.Empty(t, secrets)
	})

	t.Run("return config that is stored in cache", func(t *testing.T) {
		// given
		originalConfig := NewToolchainConfigObjWithReset(t, testconfig.AutomaticApproval().Enabled(true))
		cl := test.NewFakeClient(t, originalConfig)

		// when
		actual, secrets, err := GetConfig(cl, &toolchainv1alpha1.ToolchainConfig{})

		// then
		require.NoError(t, err)
		toolchaincfg, ok := actual.(*toolchainv1alpha1.ToolchainConfig)
		require.True(t, ok)
		assert.Equal(t, originalConfig.Spec, toolchaincfg.Spec)
		assert.Empty(t, secrets, secrets)

		t.Run("returns the same when the cache hasn't been updated", func(t *testing.T) {
			// given
			newConfig := NewToolchainConfigObjWithReset(t, testconfig.AutomaticApproval().Enabled(true))
			cl := test.NewFakeClient(t, newConfig)

			// when
			config, secrets, err := GetConfig(cl, &toolchainv1alpha1.ToolchainConfig{})

			// then
			require.NoError(t, err)

			toolchaincfg, ok := config.(*toolchainv1alpha1.ToolchainConfig)
			require.True(t, ok)
			assert.Equal(t, originalConfig.Spec, toolchaincfg.Spec)
			assert.Empty(t, secrets, secrets)
		})

		t.Run("returns the new config when the cache was updated", func(t *testing.T) {
			// given
			newConfig := NewToolchainConfigObjWithReset(t,
				testconfig.AutomaticApproval().Enabled(true),
				testconfig.Deactivation().DeactivatingNotificationDays(5),
				testconfig.Notifications().Secret().
					Ref("notification-secret").
					MailgunAPIKey("mailgunAPIKey"),
			)
			secretData := map[string]map[string]string{
				"notification-secret": {
					"mailgunAPIKey": "abc123",
				},
			}

			// when
			UpdateConfig(newConfig, secretData)

			// then
			config, secrets, err := GetConfig(cl, &toolchainv1alpha1.ToolchainConfig{})
			require.NoError(t, err)

			toolchaincfg, ok := config.(*toolchainv1alpha1.ToolchainConfig)
			require.True(t, ok)
			assert.Equal(t, newConfig.Spec, toolchaincfg.Spec)
			assert.Equal(t, secretData, secrets)
		})
	})
}

func TestGetConfigFailed(t *testing.T) {
	restore := test.SetEnvVarAndRestore(t, "WATCH_NAMESPACE", test.HostOperatorNs)
	defer restore()
	// given
	t.Run("config not found", func(t *testing.T) {
		config := NewToolchainConfigObjWithReset(t)
		cl := test.NewFakeClient(t, config)
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			return apierrors.NewNotFound(schema.GroupResource{}, "config")
		}

		// when
		actual, secrets, err := GetConfig(cl, &toolchainv1alpha1.ToolchainConfig{})

		// then
		require.NoError(t, err)
		assert.Nil(t, actual)
		assert.Empty(t, secrets)
	})

	t.Run("error getting config", func(t *testing.T) {
		config := NewToolchainConfigObjWithReset(t)
		cl := test.NewFakeClient(t, config)
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			return fmt.Errorf("some error")
		}

		// when
		actual, secrets, err := GetConfig(cl, &toolchainv1alpha1.ToolchainConfig{})

		// then
		require.Error(t, err)
		assert.Nil(t, actual)
		assert.Empty(t, secrets)
	})

	t.Run("load secrets error", func(t *testing.T) {
		config := NewToolchainConfigObjWithReset(t)
		// given
		cl := test.NewFakeClient(t, config)
		cl.MockList = func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
			return fmt.Errorf("list error")
		}

		// when
		actual, secrets, err := LoadLatest(cl, &toolchainv1alpha1.ToolchainConfig{})

		// then
		require.EqualError(t, err, "list error")
		assert.Nil(t, actual)
		assert.Empty(t, secrets)
	})
}

func TestGetCachedConfig(t *testing.T) {
	t.Run("cache empty", func(t *testing.T) {
		// when
		actual, secrets := GetCachedConfig()

		// then
		assert.Nil(t, actual)
		assert.Empty(t, secrets)
	})

	t.Run("cache filled", func(t *testing.T) {
		// given
		original := NewToolchainConfigObjWithReset(t, testconfig.AutomaticApproval().Enabled(true))

		secretData := map[string]map[string]string{
			"notification-secret": {
				"mailgunAPIKey": "abc",
			},
		}
		UpdateConfig(original, secretData)

		// when
		actual, secrets := GetCachedConfig()

		// then
		require.NotNil(t, actual)
		toolchaincfg, ok := actual.(*toolchainv1alpha1.ToolchainConfig)
		require.True(t, ok)
		assert.Equal(t, original.Spec, toolchaincfg.Spec)
		assert.Equal(t, secretData, secrets)
	})
}

func TestLoadLatest(t *testing.T) {
	restore := test.SetEnvVarAndRestore(t, "WATCH_NAMESPACE", test.HostOperatorNs)
	defer restore()
	t.Run("config found", func(t *testing.T) {
		initConfig := NewToolchainConfigObjWithReset(t, testconfig.AutomaticApproval().Enabled(true))
		initSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "notification-secret",
				Namespace: test.HostOperatorNs,
			},
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				"mailgunAPIKey": []byte("abc123"),
			},
		}
		// given
		cl := test.NewFakeClient(t, initConfig, initSecret)

		// when
		actual, secrets, err := LoadLatest(cl, &toolchainv1alpha1.ToolchainConfig{})

		// then
		require.NoError(t, err)
		toolchaincfg, ok := actual.(*toolchainv1alpha1.ToolchainConfig)
		require.True(t, ok)
		assert.True(t, *toolchaincfg.Spec.Host.AutomaticApproval.Enabled)
		assert.Len(t, secrets, 1)
		assert.Equal(t, "abc123", secrets["notification-secret"]["mailgunAPIKey"])

		t.Run("returns the same when the config hasn't been updated", func(t *testing.T) {
			// when
			actual, secrets, err := LoadLatest(cl, &toolchainv1alpha1.ToolchainConfig{})

			// then
			require.NoError(t, err)
			toolchaincfg, ok := actual.(*toolchainv1alpha1.ToolchainConfig)
			require.True(t, ok)
			assert.True(t, *toolchaincfg.Spec.Host.AutomaticApproval.Enabled)
			assert.Len(t, secrets, 1)
			assert.Equal(t, "abc123", secrets["notification-secret"]["mailgunAPIKey"])
		})

		t.Run("returns the new value when the config has been updated", func(t *testing.T) {
			// get
			changedConfig := UpdateToolchainConfigObjWithReset(t, cl, testconfig.AutomaticApproval().Enabled(false))
			err := cl.Update(context.TODO(), changedConfig)
			require.NoError(t, err)

			initSecret.Data = map[string][]byte{
				"mailgunAPIKey": []byte("abc456"),
			}
			err = cl.Update(context.TODO(), initSecret)
			require.NoError(t, err)

			// when
			actual, secrets, err := LoadLatest(cl, &toolchainv1alpha1.ToolchainConfig{})

			// then
			require.NoError(t, err)
			toolchaincfg, ok := actual.(*toolchainv1alpha1.ToolchainConfig)
			require.True(t, ok)
			assert.False(t, *toolchaincfg.Spec.Host.AutomaticApproval.Enabled)
			assert.Len(t, secrets, 1)
			assert.Equal(t, "abc456", secrets["notification-secret"]["mailgunAPIKey"])
		})
	})

	t.Run("config not found", func(t *testing.T) {
		// given
		cl := test.NewFakeClient(t)

		// when
		actual, secrets, err := LoadLatest(cl, &toolchainv1alpha1.ToolchainConfig{})

		// then
		require.NoError(t, err)
		assert.Nil(t, actual)
		assert.Empty(t, secrets)
	})

	t.Run("get config error", func(t *testing.T) {
		initconfig := NewToolchainConfigObjWithReset(t)
		// given
		cl := test.NewFakeClient(t, initconfig)
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			return fmt.Errorf("get error")
		}

		// when
		actual, secrets, err := LoadLatest(cl, &toolchainv1alpha1.ToolchainConfig{})

		// then
		require.EqualError(t, err, "get error")
		assert.Nil(t, actual)
		assert.Empty(t, secrets)
	})

	t.Run("load secrets error", func(t *testing.T) {
		initconfig := NewToolchainConfigObjWithReset(t)
		// given
		cl := test.NewFakeClient(t, initconfig)
		cl.MockList = func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
			return fmt.Errorf("list error")
		}

		// when
		actual, secrets, err := LoadLatest(cl, &toolchainv1alpha1.ToolchainConfig{})

		// then
		require.EqualError(t, err, "list error")
		assert.Nil(t, actual)
		assert.Empty(t, secrets)
	})
}

func TestMultipleExecutionsInParallel(t *testing.T) {
	restore := test.SetEnvVarAndRestore(t, "WATCH_NAMESPACE", test.HostOperatorNs)
	defer restore()
	// given
	var latch sync.WaitGroup
	latch.Add(1)
	var waitForFinished sync.WaitGroup
	initconfig := NewToolchainConfigObjWithReset(t, testconfig.Members().SpecificPerMemberCluster("member", toolchainv1alpha1.MemberOperatorConfigSpec{
		Environment: ptr.To("env"),
	}))

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "notification-secret",
			Namespace: test.HostOperatorNs,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			"mailgunAPIKey": []byte("abc0"),
		},
	}
	cl := test.NewFakeClient(t, initconfig, secret)

	for i := 0; i < 1000; i++ {
		waitForFinished.Add(2)
		go func() {
			defer waitForFinished.Done()
			latch.Wait()

			// when
			config, secrets, err := GetConfig(cl, &toolchainv1alpha1.ToolchainConfig{})

			// then
			require.NoError(t, err)
			toolchaincfg, ok := config.(*toolchainv1alpha1.ToolchainConfig)
			assert.True(t, ok)                    // go-require: require must only be used in the goroutine running the test function (testifylint)
			assert.NotEmpty(t, toolchaincfg.Spec) // go-require: require must only be used in the goroutine running the test function (testifylint)
			assert.NotEmpty(t, secrets)           //go-require: require must only be used in the goroutine running the test function (testifylint)
		}()
		go func(i int) {
			defer waitForFinished.Done()
			latch.Wait()
			config := NewToolchainConfigObjWithReset(t, testconfig.Members().SpecificPerMemberCluster(fmt.Sprintf("member%d", i), toolchainv1alpha1.MemberOperatorConfigSpec{
				Environment: ptr.To(fmt.Sprintf("env%d", i)),
			}))

			secretData := map[string]map[string]string{
				"notification-secret": {
					"mailgunAPIKey": fmt.Sprintf("abc%d", i),
				},
			}
			UpdateConfig(config, secretData)
		}(i)
	}

	// when
	latch.Done()
	waitForFinished.Wait()
	config, secrets, err := GetConfig(test.NewFakeClient(t), &toolchainv1alpha1.ToolchainConfig{})

	// then
	require.NoError(t, err)
	toolchaincfg, ok := config.(*toolchainv1alpha1.ToolchainConfig)
	require.True(t, ok)
	assert.NotEmpty(t, toolchaincfg.Spec)
	require.NotEmpty(t, secrets)
}
