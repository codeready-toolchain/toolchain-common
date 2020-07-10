package toolchaincluster

import (
	"context"
	"testing"

	"github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	nodes = []*corev1.Node{
		{
			ObjectMeta: v1.ObjectMeta{
				Name: "1a",
				Labels: map[string]string{
					corev1.LabelZoneFailureDomain: "us-east-1a",
					corev1.LabelZoneRegion:        "us-east-1",
				},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name: "1b",
				Labels: map[string]string{
					corev1.LabelZoneFailureDomain: "us-east-1b",
					corev1.LabelZoneRegion:        "us-east-1",
				},
			},
		},
	}
	zones = []string{"us-east-1a", "us-east-1b"}
)

func TestClusterHealthChecks(t *testing.T) {
	// given
	defer gock.Off()
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

	t.Run("ToolchainCluster.status doesn't contain any conditions", func(t *testing.T) {
		unstable, sec := newToolchainCluster("unstable", "http://unstable.com", v1alpha1.ToolchainClusterStatus{})
		notFound, _ := newToolchainCluster("not-found", "http://not-found.com", v1alpha1.ToolchainClusterStatus{})
		stable, _ := newToolchainCluster("stable", "http://cluster.com", v1alpha1.ToolchainClusterStatus{})

		cl := test.NewFakeClient(t, unstable, notFound, stable, sec)
		reset := setupCachedClusters(t, cl, true, unstable, notFound, stable)
		defer reset()

		// when
		err := updateClusterStatuses("test-namespace", cl)

		// then
		require.NoError(t, err)
		assertClusterStatus(t, cl, "unstable", nil, notOffline(), unhealthy())
		assertClusterStatus(t, cl, "not-found", nil, offline())
		assertClusterStatus(t, cl, "stable", nodes, healthy())
	})

	t.Run("ToolchainCluster.status already contains conditions", func(t *testing.T) {
		unstable, sec := newToolchainCluster("unstable", "http://unstable.com", withStatus(zones, healthy()))
		notFound, _ := newToolchainCluster("not-found", "http://not-found.com", withStatus(zones, notOffline(), unhealthy()))
		stable, _ := newToolchainCluster("stable", "http://cluster.com", withStatus(zones, offline()))

		cl := test.NewFakeClient(t, unstable, notFound, stable, sec)
		resetCache := setupCachedClusters(t, cl, true, unstable, notFound, stable)
		defer resetCache()

		// when
		err := updateClusterStatuses("test-namespace", cl)

		// then
		require.NoError(t, err)
		assertClusterStatus(t, cl, "unstable", nil, notOffline(), unhealthy())
		assertClusterStatus(t, cl, "not-found", nil, offline())
		assertClusterStatus(t, cl, "stable", nodes, healthy())
	})

	t.Run("if no zones nor region is retrieved, then keep the current", func(t *testing.T) {
		stable, sec := newToolchainCluster("stable", "http://cluster.com", withStatus(zones, offline()))

		cl := test.NewFakeClient(t, stable, sec)
		resetCache := setupCachedClusters(t, cl, false, stable)
		defer resetCache()

		// when
		err := updateClusterStatuses("test-namespace", cl)

		// then
		require.NoError(t, err)
		assertClusterStatus(t, cl, "stable", nodes, healthy())
	})

	t.Run("if the connection cannot be established at beginning, then it should be offline", func(t *testing.T) {
		stable, sec := newToolchainCluster("failing", "http://failing.com", v1alpha1.ToolchainClusterStatus{})

		cl := test.NewFakeClient(t, stable, sec)

		// when
		err := updateClusterStatuses("test-namespace", cl)

		// then
		require.NoError(t, err)
		assertClusterStatus(t, cl, "failing", nil, offline())
	})
}

func setupCachedClusters(t *testing.T, cl *test.FakeClient, addNodes bool, clusters ...*v1alpha1.ToolchainCluster) func() {
	service := cluster.NewToolchainClusterService(cl, logf.Log, "test-namespace", 0)
	for _, clustr := range clusters {
		err := service.AddOrUpdateToolchainCluster(clustr)
		require.NoError(t, err)
		tc, found := cluster.GetCachedToolchainCluster(clustr.Name)
		require.True(t, found)
		if addNodes {
			tc.Client = test.NewFakeClient(t, nodes[0], nodes[1])
		} else {
			tc.Client = test.NewFakeClient(t)
		}
	}
	return func() {
		for _, clustr := range clusters {
			service.DeleteToolchainCluster(clustr.Name)
		}
	}
}

func withStatus(zones []string, conditions ...v1alpha1.ToolchainClusterCondition) v1alpha1.ToolchainClusterStatus {
	region := "us-east-1"
	return v1alpha1.ToolchainClusterStatus{
		Conditions: conditions,
		Region:     &region,
		Zones:      zones,
	}
}

func newToolchainCluster(name, apiEndpoint string, status v1alpha1.ToolchainClusterStatus) (*v1alpha1.ToolchainCluster, *corev1.Secret) {
	toolchainCluster, secret := test.NewToolchainClusterWithEndpoint(name, "secret", apiEndpoint, status, map[string]string{})
	return toolchainCluster, secret
}

func assertClusterStatus(t *testing.T, cl client.Client, clusterName string, nodes []*corev1.Node, clusterConds ...v1alpha1.ToolchainClusterCondition) {
	tc := &v1alpha1.ToolchainCluster{}
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
	if nodes == nil {
		assert.Empty(t, tc.Status.Region)
		assert.Empty(t, tc.Status.Zones)
	} else {
		assert.Len(t, tc.Status.Zones, len(nodes))
		for _, node := range nodes {
			assert.Equal(t, node.Labels[corev1.LabelZoneRegion], *tc.Status.Region)
			assert.Contains(t, tc.Status.Zones, node.Labels[corev1.LabelZoneFailureDomain])
		}
	}
}

func healthy() v1alpha1.ToolchainClusterCondition {
	return v1alpha1.ToolchainClusterCondition{
		Type:    v1alpha1.ToolchainClusterReady,
		Status:  corev1.ConditionTrue,
		Reason:  "ClusterReady",
		Message: "/healthz responded with ok",
	}
}
func unhealthy() v1alpha1.ToolchainClusterCondition {
	return v1alpha1.ToolchainClusterCondition{Type: v1alpha1.ToolchainClusterReady,
		Status:  corev1.ConditionFalse,
		Reason:  "ClusterNotReady",
		Message: "/healthz responded without ok",
	}
}
func offline() v1alpha1.ToolchainClusterCondition {
	return v1alpha1.ToolchainClusterCondition{Type: v1alpha1.ToolchainClusterOffline,
		Status:  corev1.ConditionTrue,
		Reason:  "ClusterNotReachable",
		Message: "cluster is not reachable",
	}
}
func notOffline() v1alpha1.ToolchainClusterCondition {
	return v1alpha1.ToolchainClusterCondition{Type: v1alpha1.ToolchainClusterOffline,
		Status:  corev1.ConditionFalse,
		Reason:  "ClusterReachable",
		Message: "cluster is reachable",
	}
}
