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
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
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
		unstable, sec := newToolchainCluster("unstable", "http://unstable.com", toolchainv1alpha1.ToolchainClusterStatus{})
		notFound, _ := newToolchainCluster("not-found", "http://not-found.com", toolchainv1alpha1.ToolchainClusterStatus{})
		stable, _ := newToolchainCluster("stable", "http://cluster.com", toolchainv1alpha1.ToolchainClusterStatus{})

		cl := test.NewFakeClient(t, unstable, notFound, stable, sec)
		reset := setupCachedClusters(t, cl, unstable, notFound, stable)
		defer reset()

		// when
		updateClusterStatuses(context.TODO(), "test-namespace", cl)

		// then
		assertClusterStatus(t, cl, "unstable", notOffline(), unhealthy())
		assertClusterStatus(t, cl, "not-found", offline())
		assertClusterStatus(t, cl, "stable", healthy())
	})

	t.Run("ToolchainCluster.status already contains conditions", func(t *testing.T) {
		unstable, sec := newToolchainCluster("unstable", "http://unstable.com", withStatus(healthy()))
		notFound, _ := newToolchainCluster("not-found", "http://not-found.com", withStatus(notOffline(), unhealthy()))
		stable, _ := newToolchainCluster("stable", "http://cluster.com", withStatus(offline()))
		cl := test.NewFakeClient(t, unstable, notFound, stable, sec)
		resetCache := setupCachedClusters(t, cl, unstable, notFound, stable)
		defer resetCache()

		// when
		updateClusterStatuses(context.TODO(), "test-namespace", cl)

		// then
		assertClusterStatus(t, cl, "unstable", notOffline(), unhealthy())
		assertClusterStatus(t, cl, "not-found", offline())
		assertClusterStatus(t, cl, "stable", healthy())
	})

	t.Run("if no zones nor region is retrieved, then keep the current", func(t *testing.T) {
		stable, sec := newToolchainCluster("stable", "http://cluster.com", withStatus(offline()))
		cl := test.NewFakeClient(t, stable, sec)
		resetCache := setupCachedClusters(t, cl, stable)
		defer resetCache()

		// when
		updateClusterStatuses(context.TODO(), "test-namespace", cl)

		// then
		assertClusterStatus(t, cl, "stable", healthy())
	})

	t.Run("if the connection cannot be established at beginning, then it should be offline", func(t *testing.T) {
		stable, sec := newToolchainCluster("failing", "http://failing.com", toolchainv1alpha1.ToolchainClusterStatus{})

		cl := test.NewFakeClient(t, stable, sec)

		// when
		updateClusterStatuses(context.TODO(), "test-namespace", cl)

		// then
		assertClusterStatus(t, cl, "failing", offline())
	})
}

func setupCachedClusters(t *testing.T, cl *test.FakeClient, clusters ...*toolchainv1alpha1.ToolchainCluster) func() {
	service := cluster.NewToolchainClusterServiceWithClient(cl, logf.Log, "test-namespace", 0, func(config *rest.Config, options client.Options) (client.Client, error) {
		// make sure that insecure is false to make Gock mocking working properly
		config.Insecure = false
		return client.New(config, options)
	})
	for _, clustr := range clusters {
		err := service.AddOrUpdateToolchainCluster(clustr)
		require.NoError(t, err)
		tc, found := cluster.GetCachedToolchainCluster(clustr.Name)
		require.True(t, found)
		tc.Client = test.NewFakeClient(t)
	}
	return func() {
		for _, clustr := range clusters {
			service.DeleteToolchainCluster(clustr.Name)
		}
	}
}

func withStatus(conditions ...toolchainv1alpha1.ToolchainClusterCondition) toolchainv1alpha1.ToolchainClusterStatus {
	return toolchainv1alpha1.ToolchainClusterStatus{
		Conditions: conditions,
	}
}

func newToolchainCluster(name, apiEndpoint string, status toolchainv1alpha1.ToolchainClusterStatus) (*toolchainv1alpha1.ToolchainCluster, *corev1.Secret) {
	toolchainCluster, secret := test.NewToolchainClusterWithEndpoint(name, test.MemberOperatorNs, "secret", apiEndpoint, status, map[string]string{"namespace": test.MemberOperatorNs})
	return toolchainCluster, secret
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
