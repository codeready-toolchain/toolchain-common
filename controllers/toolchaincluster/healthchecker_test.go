package toolchaincluster

import (
	"context"
	"fmt"
	"testing"

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
	gock.New("https://cluster.com").
		Get("healthz").
		Persist().
		Reply(200).
		BodyString("ok")
	gock.New("https://unstable.com").
		Get("healthz").
		Persist().
		Reply(200).
		BodyString("unstable")
	gock.New("https://not-found.com").
		Get("healthz").
		Persist().
		Reply(404)

	tests := map[string]struct {
		tcType      string
		apiEndPoint string
		healthCheck bool
		err         error
	}{
		"HealthOkay": {
			tcType:      "stable",
			apiEndPoint: "https://cluster.com",
			healthCheck: true,
		},
		"HealthNotOkayButNoError": {
			tcType:      "unstable",
			apiEndPoint: "https://unstable.com",
			healthCheck: false,
		},
		"ErrorWhileDoingHealth": {
			tcType:      "Notfound",
			apiEndPoint: "https://not-found.com",
			healthCheck: false,
			err:         fmt.Errorf("the server could not find the requested resource"),
		},
	}
	for k, tc := range tests {
		t.Run(k, func(t *testing.T) {
			// given
			tcType, sec := newToolchainCluster(t, tc.tcType, tcNs, tc.apiEndPoint)
			cl := test.NewFakeClient(t, tcType, sec)
			reset := setupCachedClusters(t, cl, tcType)
			defer reset()
			cachedTC, found := cluster.GetCachedToolchainCluster(tcType.Name)
			require.True(t, found)
			cacheClient, err := kubeclientset.NewForConfig(cachedTC.RestConfig)
			require.NoError(t, err)

			// when
			healthCheck, err := getClusterHealthStatus(context.TODO(), cacheClient)

			// then
			require.Equal(t, tc.healthCheck, healthCheck)
			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
