package toolchainclusterhealth

import (
	"context"
	"fmt"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

	requeAfter := 10 * time.Second
	withCA := false

	t.Run("ToolchainCluster not found", func(t *testing.T) {
		unstable, sec := newToolchainCluster("unstable", tcNs, "http://unstable.com", toolchainv1alpha1.ToolchainClusterStatus{})

		cl := test.NewFakeClient(t, sec)
		reset := setupCachedClusters(t, cl, unstable)
		defer reset()
		service := newToolchainClusterService(t, cl, withCA)
		// given
		controller, req := prepareReconcile(unstable, cl, service, requeAfter)

		// when
		_, err := controller.Reconcile(context.TODO(), req)

		// then
		require.NoError(t, err)

	})

	t.Run("Error while getting ToolchainCluster", func(t *testing.T) {
		unstable, sec := newToolchainCluster("unstable", tcNs, "http://unstable.com", toolchainv1alpha1.ToolchainClusterStatus{})

		cl := test.NewFakeClient(t, sec)

		cl.MockGet = func(ctx context.Context, key runtimeclient.ObjectKey, obj runtimeclient.Object, opts ...runtimeclient.GetOption) error {
			if _, ok := obj.(*toolchainv1alpha1.ToolchainCluster); ok {
				return fmt.Errorf("mock error")
			}
			return cl.Client.Get(ctx, key, obj, opts...)
		}

		service := newToolchainClusterService(t, cl, withCA)
		// given
		controller, req := prepareReconcile(unstable, cl, service, requeAfter)

		// when
		_, err := controller.Reconcile(context.TODO(), req)

		// then
		require.EqualError(t, err, "mock error")

	})
	t.Run("if the connection cannot be established at beginning, then it should be offline", func(t *testing.T) {

		stable, sec := newToolchainCluster("failing", tcNs, "http://failing.com", toolchainv1alpha1.ToolchainClusterStatus{})
		cl := test.NewFakeClient(t, stable, sec)
		service := newToolchainClusterService(t, cl, withCA)
		// given
		controller, req := prepareReconcile(stable, cl, service, requeAfter)

		// when
		_, err := controller.Reconcile(context.TODO(), req)

		// then
		require.NoError(t, err)
		assertClusterStatus(t, cl, "failing", offline())
	})
	t.Run("if no zones nor region is retrieved, then keep the current", func(t *testing.T) {
		stable, sec := newToolchainCluster("stable", tcNs, "http://cluster.com", withStatus(offline()))
		cl := test.NewFakeClient(t, stable, sec)
		resetCache := setupCachedClusters(t, cl, stable)
		defer resetCache()
		service := newToolchainClusterService(t, cl, withCA)
		// given
		controller, req := prepareReconcile(stable, cl, service, requeAfter)

		// when
		_, err := controller.Reconcile(context.TODO(), req)

		// then
		require.NoError(t, err)
		assertClusterStatus(t, cl, "stable", healthy())
	})

	tests := map[string]struct {
		tctype            string
		apiendpoint       string
		clustercondition1 toolchainv1alpha1.ToolchainClusterCondition
		clustercondition2 toolchainv1alpha1.ToolchainClusterCondition
		status            toolchainv1alpha1.ToolchainClusterStatus
	}{
		"UnstableNoCondition": {
			tctype:            "unstable",
			apiendpoint:       "http://unstable.com",
			clustercondition1: notOffline(),
			clustercondition2: unhealthy(),
			status:            toolchainv1alpha1.ToolchainClusterStatus{},
		},
		"StableNoCondition": {
			tctype:            "stable",
			apiendpoint:       "http://cluster.com",
			clustercondition1: healthy(),
			status:            toolchainv1alpha1.ToolchainClusterStatus{},
		},
		"NotFoundNoCondition": {
			tctype:            "not-found",
			apiendpoint:       "http://not-found.com",
			clustercondition1: offline(),
			status:            toolchainv1alpha1.ToolchainClusterStatus{},
		},
		"UnstableContainsCondition": {
			tctype:            "unstable",
			apiendpoint:       "http://unstable.com",
			clustercondition1: notOffline(),
			clustercondition2: unhealthy(),
			status:            withStatus(healthy()),
		},
		"StableContainsCondition": {
			tctype:            "stable",
			apiendpoint:       "http://cluster.com",
			clustercondition1: healthy(),
			status:            withStatus(healthy()),
		},
		"NotFoundContainsCondition": {
			tctype:            "not-found",
			apiendpoint:       "http://not-found.com",
			clustercondition1: offline(),
			status:            withStatus(healthy()),
		},
	}
	for k, tc := range tests {
		t.Run(k, func(t *testing.T) {
			tctype, sec := newToolchainCluster(tc.tctype, tcNs, tc.apiendpoint, tc.status)
			cl := test.NewFakeClient(t, tctype, sec)
			reset := setupCachedClusters(t, cl, tctype)
			defer reset()

			service := newToolchainClusterService(t, cl, withCA)
			//given
			controller, req := prepareReconcile(tctype, cl, service, requeAfter)

			//when
			_, err := controller.Reconcile(context.TODO(), req)
			//then
			require.NoError(t, err)
			if k == "UnstableNoCondition" || k == "UnstableContainsCondition" {
				assertClusterStatus(t, cl, tc.tctype, tc.clustercondition1, tc.clustercondition2)
			} else {
				assertClusterStatus(t, cl, tc.tctype, tc.clustercondition1)
			}
		})
	}
}

func setupCachedClusters(t *testing.T, cl *test.FakeClient, clusters ...*toolchainv1alpha1.ToolchainCluster) func() {
	service := cluster.NewToolchainClusterServiceWithClient(cl, logf.Log, test.MemberOperatorNs, 0, func(config *rest.Config, options client.Options) (client.Client, error) {
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

func newToolchainCluster(name, tcNs string, apiEndpoint string, status toolchainv1alpha1.ToolchainClusterStatus) (*toolchainv1alpha1.ToolchainCluster, *corev1.Secret) {
	toolchainCluster, secret := test.NewToolchainClusterWithEndpoint(name, tcNs, "secret", apiEndpoint, status, map[string]string{"namespace": "test-namespace"})
	return toolchainCluster, secret
}

func newToolchainClusterService(t *testing.T, cl client.Client, withCA bool) cluster.ToolchainClusterService {
	return cluster.NewToolchainClusterServiceWithClient(cl, logf.Log, "test-namespace", 3*time.Second, func(config *rest.Config, options client.Options) (client.Client, error) {
		if withCA {
			assert.False(t, config.Insecure)
			assert.Equal(t, []byte("dummy"), config.CAData)
		} else {
			assert.True(t, config.Insecure)
		}
		// make sure that insecure is false to make Gock mocking working properly
		config.Insecure = false
		// reset the dummy certificate
		config.CAData = []byte("")
		return client.New(config, options)
	})
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

func prepareReconcile(toolchainCluster *toolchainv1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService, requeAfter time.Duration) (Reconciler, reconcile.Request) {
	controller := Reconciler{
		client:              cl,
		scheme:              scheme.Scheme,
		clusterCacheService: service,
		requeAfter:          requeAfter,
	}
	req := reconcile.Request{
		NamespacedName: test.NamespacedName(toolchainCluster.Namespace, toolchainCluster.Name),
	}
	return controller, req
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
