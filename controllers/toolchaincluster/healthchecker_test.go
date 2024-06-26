package toolchaincluster

import (
	"context"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
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
		tctype            string
		apiendpoint       string
		clusterconditions []toolchainv1alpha1.Condition
		status            toolchainv1alpha1.ToolchainClusterStatus
	}{
		//ToolchainCluster.status doesn't contain any conditions
		"UnstableNoCondition": {
			tctype:            "unstable",
			apiendpoint:       "http://unstable.com",
			clusterconditions: []toolchainv1alpha1.Condition{unhealthy(), notOffline()},
		},
		"StableNoCondition": {
			tctype:            "stable",
			apiendpoint:       "http://cluster.com",
			clusterconditions: []toolchainv1alpha1.Condition{healthy()},
		},
		"NotFoundNoCondition": {
			tctype:            "not-found",
			apiendpoint:       "http://not-found.com",
			clusterconditions: []toolchainv1alpha1.Condition{offline()},
		},
		//ToolchainCluster.status already contains conditions
		"UnstableContainsCondition": {
			tctype:            "unstable",
			apiendpoint:       "http://unstable.com",
			clusterconditions: []toolchainv1alpha1.Condition{unhealthy(), notOffline()},
		},
		"StableContainsCondition": {
			tctype:            "stable",
			apiendpoint:       "http://cluster.com",
			clusterconditions: []toolchainv1alpha1.Condition{healthy()},
		},
		"NotFoundContainsCondition": {
			tctype:            "not-found",
			apiendpoint:       "http://not-found.com",
			clusterconditions: []toolchainv1alpha1.Condition{offline()},
		},
		//if the connection cannot be established at beginning, then it should be offline
		"OfflineConnectionNotEstablished": {
			tctype:            "failing",
			apiendpoint:       "http://failing.com",
			clusterconditions: []toolchainv1alpha1.Condition{offline()},
		},
		//if no zones nor region is retrieved, then keep the current
		"NoZoneKeepCurrent": {
			tctype:            "stable",
			apiendpoint:       "http://cluster.com",
			clusterconditions: []toolchainv1alpha1.Condition{healthy()},
		},
	}
	for k, tc := range tests {
		t.Run(k, func(t *testing.T) {
			tctype, sec := newToolchainCluster(tc.tctype, tcNs, tc.apiendpoint, tc.status)
			cl := test.NewFakeClient(t, tctype, sec)
			reset := setupCachedClusters(t, cl, tctype)
			defer reset()
			cachedtc, found := cluster.GetCachedToolchainCluster(tctype.Name)
			require.True(t, found)
			cacheclient, err := kubeclientset.NewForConfig(cachedtc.RestConfig)
			require.NoError(t, err)

			//when
			hcond := GetClusterHealthStatus(context.TODO(), cacheclient)

			//then
			assert.Len(t, tc.clusterconditions, len(hcond))
		SetConditions:
			for _, hc := range hcond {
				for _, tco := range tc.clusterconditions {
					if hc.Type == tco.Type {
						assert.Equal(t, tco.Status, hc.Status)
						assert.Equal(t, tco.Reason, hc.Reason)
						assert.Equal(t, tco.Message, hc.Message)
						continue SetConditions
					}
				}
				assert.Failf(t, "condition not found", "the list of conditions %v doesn't contain the expected condition %v", tc.clusterconditions, hc)
			}

		})
	}
}

func healthy() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionTrue,
		Reason:  "ClusterReady",
		Message: "/healthz responded with ok",
	}
}
func unhealthy() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{Type: toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  "ClusterNotReady",
		Message: "/healthz responded without ok",
	}
}
func offline() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{Type: toolchainv1alpha1.ToolchainClusterOffline,
		Status:  corev1.ConditionTrue,
		Reason:  "ClusterNotReachable",
		Message: "cluster is not reachable",
	}
}
func notOffline() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{Type: toolchainv1alpha1.ToolchainClusterOffline,
		Status:  corev1.ConditionFalse,
		Reason:  "ClusterReachable",
		Message: "cluster is reachable",
	}
}
