package toolchaincluster

import (
	"context"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	kubeclientset "k8s.io/client-go/kubernetes"
)

func TestClusterHealthChecks(t *testing.T) {

	// given
	defer gock.Off()
	tcNs := "test-namespace"
	gock.New("http://cluster.com").
		Get("healthz").
		Persist().
		Reply(200).
		BodyString("ok")
	gock.New("http://unstable.com").
		Get("healthz").
		Persist().
		Reply(200).
		BodyString("unstable")
	gock.New("http://not-found.com").
		Get("healthz").
		Persist().
		Reply(404)

	t.Run("When cluster health is ok", func(t *testing.T) {
		tctype, sec := newToolchainCluster("stable", tcNs, "http://cluster.com", toolchainv1alpha1.ToolchainClusterStatus{})
		cl := test.NewFakeClient(t, tctype, sec)
		reset := setupCachedClusters(t, cl, tctype)
		defer reset()
		cachedtc, found := cluster.GetCachedToolchainCluster(tctype.Name)
		require.True(t, found)
		cacheclient, err := kubeclientset.NewForConfig(cachedtc.RestConfig)
		require.NoError(t, err)

		//when
		health, errh := GetClusterHealth(context.TODO(), cacheclient)

		//then
		require.NoError(t, errh)
		require.Equal(t, true, health)

	})
	t.Run("When cluster health is Not ok but no error", func(t *testing.T) {
		tctype, sec := newToolchainCluster("unstable", tcNs, "http://unstable.com", toolchainv1alpha1.ToolchainClusterStatus{})
		cl := test.NewFakeClient(t, tctype, sec)
		reset := setupCachedClusters(t, cl, tctype)
		defer reset()
		cachedtc, found := cluster.GetCachedToolchainCluster(tctype.Name)
		require.True(t, found)
		cacheclient, err := kubeclientset.NewForConfig(cachedtc.RestConfig)
		require.NoError(t, err)

		//when
		health, errh := GetClusterHealth(context.TODO(), cacheclient)

		//then
		require.NoError(t, errh)
		require.Equal(t, false, health)

	})

	t.Run("Error while doing cluster health", func(t *testing.T) {
		tctype, sec := newToolchainCluster("Notfound", tcNs, "http://not-found.com", toolchainv1alpha1.ToolchainClusterStatus{})
		cl := test.NewFakeClient(t, tctype, sec)
		reset := setupCachedClusters(t, cl, tctype)
		defer reset()
		cachedtc, found := cluster.GetCachedToolchainCluster(tctype.Name)
		require.True(t, found)
		cacheclient, err := kubeclientset.NewForConfig(cachedtc.RestConfig)
		require.NoError(t, err)

		//when
		health, errh := GetClusterHealth(context.TODO(), cacheclient)

		//then
		require.Error(t, errh)
		require.Equal(t, false, health)

	})
}
