package cluster_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/codeready-toolchain/toolchain-common/pkg/test/verify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAddToolchainClusterAsMember(t *testing.T) {
	// given & then
	verify.AddToolchainClusterAsMember(t, func(toolchainCluster *toolchainv1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// when
		return service.AddOrUpdateToolchainCluster(toolchainCluster)
	})
}

func TestAddToolchainClusterAsHost(t *testing.T) {
	// given & then
	verify.AddToolchainClusterAsHost(t, func(toolchainCluster *toolchainv1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// when
		return service.AddOrUpdateToolchainCluster(toolchainCluster)
	})
}

func TestAddToolchainClusterFailsBecauseOfMissingSecret(t *testing.T) {
	// given & then
	verify.AddToolchainClusterFailsBecauseOfMissingSecret(t, func(toolchainCluster *toolchainv1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// when
		return service.AddOrUpdateToolchainCluster(toolchainCluster)
	})
}

func TestAddToolchainClusterFailsBecauseOfEmptySecret(t *testing.T) {
	// given & then
	verify.AddToolchainClusterFailsBecauseOfEmptySecret(t, func(toolchainCluster *toolchainv1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// when
		return service.AddOrUpdateToolchainCluster(toolchainCluster)
	})
}

func TestUpdateToolchainCluster(t *testing.T) {
	// given & then
	verify.UpdateToolchainCluster(t, func(toolchainCluster *toolchainv1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// when
		return service.AddOrUpdateToolchainCluster(toolchainCluster)
	})
}

func TestDeleteToolchainClusterWhenDoesNotExist(t *testing.T) {
	// given & then
	verify.DeleteToolchainCluster(t, func(toolchainCluster *toolchainv1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// when
		service.DeleteToolchainCluster("east")
		return nil
	})
}

func TestListToolchainClusterConfigs(t *testing.T) {
	// given
	status := test.NewClusterStatus(toolchainv1alpha1.ConditionReady, corev1.ConditionTrue)
	require.NoError(t, toolchainv1alpha1.AddToScheme(scheme.Scheme))

	m1, sec1 := test.NewToolchainClusterWithEndpoint(t, "east", test.HostOperatorNs, test.MemberOperatorNs, "secret1", "https://m1.com", status, false)
	m2, sec2 := test.NewToolchainClusterWithEndpoint(t, "west", test.HostOperatorNs, test.MemberOperatorNs, "secret2", "https://m2.com", status, false)
	host, secHost := test.NewToolchainCluster(t, "host", test.MemberOperatorNs, test.HostOperatorNs, "secretHost", status, false)
	noise, secNoise := test.NewToolchainCluster(t, "noise", "noise-namespace", "secretNoise", test.MemberOperatorNs, status, false)
	require.NoError(t, toolchainv1alpha1.AddToScheme(scheme.Scheme))

	cl := test.NewFakeClient(t, m1, m2, host, noise, sec1, sec2, secHost, secNoise)

	t.Run("list members", func(t *testing.T) {
		// when
		clusterConfigs, err := cluster.ListToolchainClusterConfigs(cl, m1.Namespace, time.Second)

		// then
		require.NoError(t, err)
		require.Len(t, clusterConfigs, 2)
		verify.AssertClusterConfigThat(t, clusterConfigs[0]).
			HasName("east").
			HasOperatorNamespace("toolchain-member-operator").
			HasAPIEndpoint("https://m1.com").
			RestConfigHasHost("https://m1.com")
		verify.AssertClusterConfigThat(t, clusterConfigs[1]).
			HasName("west").
			HasOperatorNamespace("toolchain-member-operator").
			HasAPIEndpoint("https://m2.com").
			RestConfigHasHost("https://m2.com")
	})

	t.Run("list host", func(t *testing.T) {
		// when

		clusterConfigs, err := cluster.ListToolchainClusterConfigs(cl, host.Namespace, time.Second)

		// then
		require.NoError(t, err)
		require.Len(t, clusterConfigs, 1)

		verify.AssertClusterConfigThat(t, clusterConfigs[0]).
			HasName("host").
			HasOperatorNamespace("toolchain-host-operator").
			HasAPIEndpoint("https://cluster.com").
			RestConfigHasHost("https://cluster.com")
	})

	t.Run("list members when there is none present", func(t *testing.T) {
		// given
		cl := test.NewFakeClient(t, host, noise, secNoise)

		// when
		clusterConfigs, err := cluster.ListToolchainClusterConfigs(cl, m1.Namespace, time.Second)

		// then
		require.NoError(t, err)
		require.Empty(t, clusterConfigs)
	})

	t.Run("when list fails", func(t *testing.T) {
		// given
		cl := test.NewFakeClient(t, m1, m2, host, noise, sec1, sec2, secHost, secNoise)
		cl.MockList = func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
			return fmt.Errorf("some error")
		}

		// when
		clusterConfigs, err := cluster.ListToolchainClusterConfigs(cl, m1.Namespace, time.Second)

		// then
		require.Error(t, err)
		require.Empty(t, clusterConfigs)
	})

	t.Run("when get secret fails", func(t *testing.T) {
		// given
		cl := test.NewFakeClient(t, m1, m2, host, noise, sec1, sec2, secHost, secNoise)
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			return fmt.Errorf("some error")
		}

		// when
		clusterConfigs, err := cluster.ListToolchainClusterConfigs(cl, m1.Namespace, time.Second)

		// then
		require.Error(t, err)
		require.Empty(t, clusterConfigs)
	})
}

func TestNewClusterConfig(t *testing.T) {
	tc := func() *toolchainv1alpha1.ToolchainCluster {
		return &toolchainv1alpha1.ToolchainCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tc",
				Namespace: "ns",
			},
			Spec: toolchainv1alpha1.ToolchainClusterSpec{
				SecretRef: toolchainv1alpha1.LocalSecretReference{
					Name: "secret",
				},
			},
		}
	}

	kubeconfigSecret := func(t *testing.T) *corev1.Secret {
		t.Helper()
		kubeconfig := clientcmdapi.Config{
			Clusters: map[string]*clientcmdapi.Cluster{
				"cluster": {
					Server: "https://over.the.rainbow",
				},
			},
			Contexts: map[string]*clientcmdapi.Context{
				"ctx": {
					Cluster:   "cluster",
					AuthInfo:  "auth",
					Namespace: "operatorns",
				},
			},
			AuthInfos: map[string]*clientcmdapi.AuthInfo{
				"auth": {
					Token: "token",
				},
			},
			CurrentContext: "ctx",
		}
		kubeconfigContents, err := clientcmd.Write(kubeconfig)
		require.NoError(t, err)

		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret",
				Namespace: "ns",
			},
			Data: map[string][]byte{
				"kubeconfig": kubeconfigContents,
			},
		}
	}

	t.Run("using kubeconfig in secret", func(t *testing.T) {
		tc := tc()
		secret := kubeconfigSecret(t)

		cl := test.NewFakeClient(t, tc, secret)

		cfg, err := cluster.NewClusterConfig(cl, tc, 1*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "https://over.the.rainbow", cfg.APIEndpoint)
		assert.Equal(t, "operatorns", cfg.OperatorNamespace)
		assert.Equal(t, "token", cfg.RestConfig.BearerToken)
	})
}
