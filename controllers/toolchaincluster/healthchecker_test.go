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
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("toolchaincluster_healthcheck")

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
		clusterconditions []toolchainv1alpha1.ToolchainClusterCondition
		status            toolchainv1alpha1.ToolchainClusterStatus
	}{
		//ToolchainCluster.status doesn't contain any conditions
		"UnstableNoCondition": {
			tctype:            "unstable",
			apiendpoint:       "http://unstable.com",
			clusterconditions: []toolchainv1alpha1.ToolchainClusterCondition{unhealthy(), notOffline()},
			status:            toolchainv1alpha1.ToolchainClusterStatus{},
		},
		"StableNoCondition": {
			tctype:            "stable",
			apiendpoint:       "http://cluster.com",
			clusterconditions: []toolchainv1alpha1.ToolchainClusterCondition{healthy()},
			status:            toolchainv1alpha1.ToolchainClusterStatus{},
		},
		"NotFoundNoCondition": {
			tctype:            "not-found",
			apiendpoint:       "http://not-found.com",
			clusterconditions: []toolchainv1alpha1.ToolchainClusterCondition{offline()},
			status:            toolchainv1alpha1.ToolchainClusterStatus{},
		},
		//ToolchainCluster.status already contains conditions
		"UnstableContainsCondition": {
			tctype:            "unstable",
			apiendpoint:       "http://unstable.com",
			clusterconditions: []toolchainv1alpha1.ToolchainClusterCondition{unhealthy(), notOffline()},
			status:            withStatus(healthy()),
		},
		"StableContainsCondition": {
			tctype:            "stable",
			apiendpoint:       "http://cluster.com",
			clusterconditions: []toolchainv1alpha1.ToolchainClusterCondition{healthy()},
			status:            withStatus(healthy()),
		},
		"NotFoundContainsCondition": {
			tctype:            "not-found",
			apiendpoint:       "http://not-found.com",
			clusterconditions: []toolchainv1alpha1.ToolchainClusterCondition{offline()},
			status:            withStatus(healthy()),
		},
		//if the connection cannot be established at beginning, then it should be offline
		"OfflineConnectionNotEstablished": {
			tctype:            "failing",
			apiendpoint:       "http://failing.com",
			clusterconditions: []toolchainv1alpha1.ToolchainClusterCondition{offline()},
			status:            toolchainv1alpha1.ToolchainClusterStatus{},
		},
		//if no zones nor region is retrieved, then keep the current
		"NoZoneKeepCurrent": {
			tctype:            "stable",
			apiendpoint:       "http://cluster.com",
			clusterconditions: []toolchainv1alpha1.ToolchainClusterCondition{healthy()},
			status:            withStatus(offline()),
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
			healthChecker := &HealthChecker{
				localClusterClient:     cl,
				remoteClusterClient:    cachedtc.Client,
				remoteClusterClientset: cacheclient,
				logger:                 logger,
			}
			// when
			err = healthChecker.updateIndividualClusterStatus(context.TODO(), tctype)

			//then
			require.NoError(t, err)
			assertClusterStatus(t, cl, tc.tctype, tc.clusterconditions...)
		})
	}
}

func withStatus(conditions ...toolchainv1alpha1.ToolchainClusterCondition) toolchainv1alpha1.ToolchainClusterStatus {
	return toolchainv1alpha1.ToolchainClusterStatus{
		Conditions: conditions,
	}
}
func assertClusterStatus(t *testing.T, cl client.Client, clusterName string, clusterConds ...toolchainv1alpha1.ToolchainClusterCondition) {
	tc := &toolchainv1alpha1.ToolchainCluster{}
	err := cl.Get(context.TODO(), test.NamespacedName("test-namespace", clusterName), tc)
	require.NoError(t, err)
	assert.Len(t, tc.Status.Conditions, len(clusterConds))
ExpConditions:
	for _, expCond := range clusterConds {
		for _, cond := range tc.Status.Conditions {
			if expCond.Type == cond.Type {
				assert.Equal(t, expCond.Status, cond.Status)
				assert.Equal(t, expCond.Reason, cond.Reason)
				assert.Equal(t, expCond.Message, cond.Message)
				continue ExpConditions
			}
		}
		assert.Failf(t, "condition not found", "the list of conditions %v doesn't contain the expected condition %v", tc.Status.Conditions, expCond)
	}
}
func healthy() toolchainv1alpha1.ToolchainClusterCondition {
	return toolchainv1alpha1.ToolchainClusterCondition{
		Type:    toolchainv1alpha1.ToolchainClusterReady,
		Status:  corev1.ConditionTrue,
		Reason:  "ClusterReady",
		Message: "/healthz responded with ok",
	}
}
func unhealthy() toolchainv1alpha1.ToolchainClusterCondition {
	return toolchainv1alpha1.ToolchainClusterCondition{Type: toolchainv1alpha1.ToolchainClusterReady,
		Status:  corev1.ConditionFalse,
		Reason:  "ClusterNotReady",
		Message: "/healthz responded without ok",
	}
}
func offline() toolchainv1alpha1.ToolchainClusterCondition {
	return toolchainv1alpha1.ToolchainClusterCondition{Type: toolchainv1alpha1.ToolchainClusterOffline,
		Status:  corev1.ConditionTrue,
		Reason:  "ClusterNotReachable",
		Message: "cluster is not reachable",
	}
}
func notOffline() toolchainv1alpha1.ToolchainClusterCondition {
	return toolchainv1alpha1.ToolchainClusterCondition{Type: toolchainv1alpha1.ToolchainClusterOffline,
		Status:  corev1.ConditionFalse,
		Reason:  "ClusterReachable",
		Message: "cluster is reachable",
	}
}
