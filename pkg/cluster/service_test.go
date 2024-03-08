package cluster_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/codeready-toolchain/toolchain-common/pkg/test/verify"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
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
	status := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionTrue)
	require.NoError(t, toolchainv1alpha1.AddToScheme(scheme.Scheme))

	m1, sec1 := test.NewToolchainClusterWithEndpoint("east", test.HostOperatorNs, "secret1", "http://m1.com", status, verify.Labels(test.MemberOperatorNs, "m1ClusterName"))
	m2, sec2 := test.NewToolchainClusterWithEndpoint("west", test.HostOperatorNs, "secret2", "http://m2.com", status, verify.Labels(test.MemberOperatorNs, "m2ClusterName"))
	host, secHost := test.NewToolchainCluster("host", test.MemberOperatorNs, "secretHost", status, verify.Labels(test.HostOperatorNs, "hostClusterName"))
	noise, secNoise := test.NewToolchainCluster("noise", "noise-namespace", "secretNoise", status, verify.Labels(test.MemberOperatorNs, "noiseClusterName"))
	require.NoError(t, toolchainv1alpha1.AddToScheme(scheme.Scheme))
	m1.Labels[cluster.RoleLabel(cluster.Tenant)] = ""
	m2.Labels[cluster.RoleLabel(cluster.Tenant)] = ""
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
			HasOwnerClusterName("m1ClusterName").
			HasAPIEndpoint("http://m1.com").
			ContainsLabel(cluster.RoleLabel(cluster.Tenant)). // the value is not used only the key matters
			RestConfigHasHost("http://m1.com")
		verify.AssertClusterConfigThat(t, clusterConfigs[1]).
			HasName("west").
			HasOperatorNamespace("toolchain-member-operator").
			HasOwnerClusterName("m2ClusterName").
			HasAPIEndpoint("http://m2.com").
			ContainsLabel(cluster.RoleLabel(cluster.Tenant)). // the value is not used only the key matters
			RestConfigHasHost("http://m2.com")
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
			HasOwnerClusterName("hostClusterName").
			HasAPIEndpoint("http://cluster.com").
			RestConfigHasHost("http://cluster.com")
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
		//given
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
		//given
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
