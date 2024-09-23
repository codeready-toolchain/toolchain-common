package toolchaincluster

import (
	"context"
	"fmt"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var requeAfter = 10 * time.Second

func TestClusterControllerChecks(t *testing.T) {
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

	t.Run("ToolchainCluster not found", func(t *testing.T) {
		// given
		NotFound, sec := newToolchainCluster(t, "notfound", tcNs, "http://not-found.com")

		cl := test.NewFakeClient(t, sec)
		reset := setupCachedClusters(t, cl, NotFound)
		defer reset()
		controller, req := prepareReconcile(NotFound, cl, requeAfter)

		// when
		recResult, err := controller.Reconcile(context.TODO(), req)

		// then
		require.NoError(t, err)
		require.Equal(t, reconcile.Result{Requeue: false, RequeueAfter: 0}, recResult)
	})

	t.Run("Error while getting ToolchainCluster", func(t *testing.T) {
		// given
		tc, sec := newToolchainCluster(t, "tc", tcNs, "http://tc.com")

		cl := test.NewFakeClient(t, sec)
		cl.MockGet = func(ctx context.Context, key runtimeclient.ObjectKey, obj runtimeclient.Object, opts ...runtimeclient.GetOption) error {
			if _, ok := obj.(*toolchainv1alpha1.ToolchainCluster); ok {
				return fmt.Errorf("mock error")
			}
			return cl.Client.Get(ctx, key, obj, opts...)
		}
		controller, req := prepareReconcile(tc, cl, requeAfter)

		// when
		recResult, err := controller.Reconcile(context.TODO(), req)

		// then
		require.EqualError(t, err, "mock error")
		require.Equal(t, reconcile.Result{Requeue: false, RequeueAfter: 0}, recResult)
	})

	t.Run("reconcile successful and requeued", func(t *testing.T) {
		// given
		stable, sec := newToolchainCluster(t, "stable", tcNs, "https://cluster.com")

		cl := test.NewFakeClient(t, stable, sec)
		reset := setupCachedClusters(t, cl, stable)

		defer reset()
		controller, req := prepareReconcile(stable, cl, requeAfter)

		// when
		recResult, err := controller.Reconcile(context.TODO(), req)

		// then
		require.NoError(t, err)
		require.Equal(t, reconcile.Result{RequeueAfter: requeAfter}, recResult)
		assertClusterStatus(t, cl, "stable", clusterReadyCondition())
	})

	t.Run("toolchain cluster cache not found", func(t *testing.T) {
		// given
		unstable, _ := newToolchainCluster(t, "unstable", tcNs, "http://unstable.com")

		cl := test.NewFakeClient(t, unstable)
		controller, req := prepareReconcile(unstable, cl, requeAfter)

		// when
		_, err := controller.Reconcile(context.TODO(), req)

		// then
		require.EqualError(t, err, "cluster unstable not found in cache")
		assertClusterStatus(t, cl, "unstable", clusterOfflineCondition("cluster unstable not found in cache"))
	})

	t.Run("error while updating a toolchain cluster status on cache not found", func(t *testing.T) {
		// given
		stable, _ := newToolchainCluster(t, "stable", tcNs, "https://cluster.com")

		cl := test.NewFakeClient(t, stable)
		cl.MockStatusUpdate = func(ctx context.Context, obj runtimeclient.Object, opts ...runtimeclient.SubResourceUpdateOption) error {
			return fmt.Errorf("mock error")
		}
		controller, req := prepareReconcile(stable, cl, requeAfter)

		// when
		recResult, err := controller.Reconcile(context.TODO(), req)

		// then
		require.EqualError(t, err, "cluster stable not found in cache")
		require.Equal(t, reconcile.Result{}, recResult)

		assertClusterStatus(t, cl, "stable")
	})

	t.Run("error while updating a toolchain cluster status when health-check failed", func(t *testing.T) {
		// given
		stable, sec := newToolchainCluster(t, "stable", tcNs, "https://cluster.com")
		expectedErr := fmt.Errorf("my test error")
		cl := test.NewFakeClient(t, stable, sec)
		cl.MockStatusUpdate = func(ctx context.Context, obj runtimeclient.Object, opts ...runtimeclient.SubResourceUpdateOption) error {
			return expectedErr
		}
		reset := setupCachedClusters(t, cl, stable)

		defer reset()
		controller, req := prepareReconcile(stable, cl, requeAfter)
		controller.checkHealth = func(context.Context, *kubeclientset.Clientset) (bool, error) {
			return false, expectedErr
		}
		// when
		recResult, err := controller.Reconcile(context.TODO(), req)

		// then
		require.EqualError(t, err, fmt.Sprintf("failed to update the status of cluster - %s: %v", stable.Name, expectedErr))
		require.Equal(t, reconcile.Result{}, recResult)
		assertClusterStatus(t, cl, "stable")
	})
}

func TestGetClusterHealth(t *testing.T) {
	t.Run("Check health default", func(t *testing.T) {
		// given
		stable, sec := newToolchainCluster(t, "stable", "test-namespace", "https://cluster.com")

		cl := test.NewFakeClient(t, stable, sec)
		reset := setupCachedClusters(t, cl, stable)

		defer reset()
		controller, req := prepareReconcile(stable, cl, requeAfter)
		controller.checkHealth = func(context.Context, *kubeclientset.Clientset) (bool, error) {
			return true, nil
		}

		// when
		recResult, err := controller.Reconcile(context.TODO(), req)

		// then
		require.NoError(t, err)
		require.Equal(t, reconcile.Result{RequeueAfter: requeAfter}, recResult)
		assertClusterStatus(t, cl, "stable", clusterReadyCondition())
	})
	t.Run("get health condition when health obtained is false ", func(t *testing.T) {
		// given
		stable, sec := newToolchainCluster(t, "stable", "test-namespace", "https://cluster.com")

		cl := test.NewFakeClient(t, stable, sec)
		reset := setupCachedClusters(t, cl, stable)

		defer reset()
		controller, req := prepareReconcile(stable, cl, requeAfter)
		controller.checkHealth = func(context.Context, *kubeclientset.Clientset) (bool, error) {
			return false, nil
		}

		// when
		recResult, err := controller.Reconcile(context.TODO(), req)

		// then
		require.NoError(t, err)
		require.Equal(t, reconcile.Result{RequeueAfter: requeAfter}, recResult)
		assertClusterStatus(t, cl, "stable", clusterNotReadyCondition())
	})
}

func setupCachedClusters(t *testing.T, cl *test.FakeClient, clusters ...*toolchainv1alpha1.ToolchainCluster) func() {
	service := cluster.NewToolchainClusterServiceWithClient(cl, logf.Log, test.MemberOperatorNs, 0, func(config *rest.Config, options runtimeclient.Options) (runtimeclient.Client, error) {
		// make sure that insecure is false to make Gock mocking working properly
		config.Insecure = false
		return runtimeclient.New(config, options)
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

func newToolchainCluster(t *testing.T, name, tcNs string, apiEndpoint string) (*toolchainv1alpha1.ToolchainCluster, *corev1.Secret) {
	toolchainCluster, secret := test.NewToolchainClusterWithEndpoint(t, name, tcNs, "test-namespace", "secret", apiEndpoint, toolchainv1alpha1.ToolchainClusterStatus{}, false)
	return toolchainCluster, secret
}

func prepareReconcile(toolchainCluster *toolchainv1alpha1.ToolchainCluster, cl *test.FakeClient, requeAfter time.Duration) (Reconciler, reconcile.Request) {
	controller := Reconciler{
		Client:     cl,
		Scheme:     scheme.Scheme,
		RequeAfter: requeAfter,
	}
	req := reconcile.Request{
		NamespacedName: test.NamespacedName(toolchainCluster.Namespace, toolchainCluster.Name),
	}
	return controller, req
}

func assertClusterStatus(t *testing.T, cl runtimeclient.Client, clusterName string, clusterConds ...toolchainv1alpha1.Condition) {
	tc := &toolchainv1alpha1.ToolchainCluster{}
	err := cl.Get(context.TODO(), test.NamespacedName("test-namespace", clusterName), tc)
	require.NoError(t, err)
	test.AssertConditionsMatch(t, tc.Status.Conditions, clusterConds...)
}
