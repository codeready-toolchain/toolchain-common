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

	tests := map[string]struct {
		tctype      string
		apiendpoint string
		healthcheck bool
		errh        error
	}{
		"HealthOkay": {
			tctype:      "stable",
			apiendpoint: "http://cluster.com",
			healthcheck: true,
			errh:        nil,
		},
		"HealthNotOkayButNoError": {
			tctype:      "unstable",
			apiendpoint: "http://unstable.com",
			healthcheck: false,
			errh:        nil,
		},
		"ErrorWhileDoingHealth": {
			tctype:      "Notfound",
			apiendpoint: "http://not-found.com",
			healthcheck: false,
		},
	}
	for k, tc := range tests {
		t.Run(k, func(t *testing.T) {
			//given
			tctype, sec := newToolchainCluster(tc.tctype, tcNs, tc.apiendpoint, toolchainv1alpha1.ToolchainClusterStatus{})
			cl := test.NewFakeClient(t, tctype, sec)
			reset := setupCachedClusters(t, cl, tctype)
			defer reset()
			cachedtc, found := cluster.GetCachedToolchainCluster(tctype.Name)
			require.True(t, found)
			cacheclient, err := kubeclientset.NewForConfig(cachedtc.RestConfig)
			require.NoError(t, err)

			//when
			healthcheck, errh := getClusterHealthStatus(context.TODO(), cacheclient)

			//then
			require.Equal(t, tc.healthcheck, healthcheck)
			if tc.tctype == "Notfound" {
				require.Error(t, errh)
			} else {
				require.Equal(t, tc.errh, errh)
			}

		})
	}
}
