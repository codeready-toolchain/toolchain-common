package toolchaincluster

import (
	"context"
	"fmt"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	"gotest.tools/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var requeAfter = 10 * time.Second

func TestClusterControllerChecks(t *testing.T) {
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

	t.Run("ToolchainCluster not found", func(t *testing.T) {
		// given
		NotFound, sec := newToolchainCluster("notfound", tcNs, "http://not-found.com", toolchainv1alpha1.ToolchainClusterStatus{})

		cl := test.NewFakeClient(t, sec)
		reset := setupCachedClusters(t, cl, NotFound)
		defer reset()
		controller, req := prepareReconcile(NotFound, cl, requeAfter)

		// when
		recresult, err := controller.Reconcile(context.TODO(), req)

		// then
		require.Equal(t, err, nil)
		require.Equal(t, reconcile.Result{Requeue: false, RequeueAfter: 0}, recresult)
	})

	t.Run("Error while getting ToolchainCluster", func(t *testing.T) {
		// given
		tc, sec := newToolchainCluster("tc", tcNs, "http://tc.com", toolchainv1alpha1.ToolchainClusterStatus{})

		cl := test.NewFakeClient(t, sec)

		cl.MockGet = func(ctx context.Context, key runtimeclient.ObjectKey, obj runtimeclient.Object, opts ...runtimeclient.GetOption) error {
			if _, ok := obj.(*toolchainv1alpha1.ToolchainCluster); ok {
				return fmt.Errorf("mock error")
			}
			return cl.Client.Get(ctx, key, obj, opts...)
		}

		controller, req := prepareReconcile(tc, cl, requeAfter)

		// when
		recresult, err := controller.Reconcile(context.TODO(), req)

		// then
		require.EqualError(t, err, "mock error")
		require.Equal(t, reconcile.Result{Requeue: false, RequeueAfter: 0}, recresult)
	})

	t.Run("reconcile successful and requeued", func(t *testing.T) {
		// given
		stable, sec := newToolchainCluster("stable", tcNs, "http://cluster.com", toolchainv1alpha1.ToolchainClusterStatus{})

		cl := test.NewFakeClient(t, stable, sec)
		reset := setupCachedClusters(t, cl, stable)
		defer reset()
		controller, req := prepareReconcile(stable, cl, requeAfter)

		// when
		recresult, err := controller.Reconcile(context.TODO(), req)

		// then
		require.Equal(t, err, nil)
		require.Equal(t, reconcile.Result{RequeueAfter: requeAfter}, recresult)
		assertClusterStatus(t, cl, "stable", healthy())
	})

	t.Run("toolchain cluster cache not found", func(t *testing.T) {
		// given
		stable, _ := newToolchainCluster("stable", tcNs, "http://cluster.com", toolchainv1alpha1.ToolchainClusterStatus{})

		cl := test.NewFakeClient(t, stable)

		controller, req := prepareReconcile(stable, cl, requeAfter)

		// when
		_, err := controller.Reconcile(context.TODO(), req)

		// then
		require.EqualError(t, err, "cluster stable not found in cache")
		actualtoolchaincluster := &toolchainv1alpha1.ToolchainCluster{}
		err = cl.Client.Get(context.TODO(), types.NamespacedName{Name: "stable", Namespace: tcNs}, actualtoolchaincluster)
		require.NoError(t, err)
		assertClusterStatus(t, cl, "stable", offline())
	})

	t.Run("migrates connection settings to kubeconfig in secret", func(t *testing.T) {
		// given
		tc, secret := newToolchainCluster("tc", tcNs, "http://cluster.com", toolchainv1alpha1.ToolchainClusterStatus{})
		cl := test.NewFakeClient(t, tc, secret)
		reset := setupCachedClusters(t, cl, tc)
		defer reset()

		controller, req := prepareReconcile(tc, cl, requeAfter)
		expectedKubeConfig := composeKubeConfigFromData([]byte("mycooltoken"), "http://cluster.com", "test-namespace", true)

		// when
		_, err := controller.Reconcile(context.TODO(), req)
		secretAfterReconcile := &corev1.Secret{}
		require.NoError(t, cl.Get(context.TODO(), client.ObjectKeyFromObject(secret), secretAfterReconcile))
		actualKubeConfig, loadErr := clientcmd.Load(secretAfterReconcile.Data["kubeconfig"])

		// then
		require.NoError(t, err)
		require.NoError(t, loadErr)
		assert.Contains(t, secretAfterReconcile.Data, "kubeconfig")

		// we need to use this more complex equals, because we don't initialize the Extension fields (i.e. they're nil)
		// while they're initialized and empty after deserialization, which causes the "normal" deep equals to fail.
		result := cmp.DeepEqual(expectedKubeConfig, *actualKubeConfig,
			cmpopts.IgnoreFields(clientcmdapi.Config{}, "Extensions"),
			cmpopts.IgnoreFields(clientcmdapi.Preferences{}, "Extensions"),
			cmpopts.IgnoreFields(clientcmdapi.Cluster{}, "Extensions"),
			cmpopts.IgnoreFields(clientcmdapi.AuthInfo{}, "Extensions"),
			cmpopts.IgnoreFields(clientcmdapi.Context{}, "Extensions"),
		)()

		assert.True(t, result.Success())
	})
}

func TestComposeKubeConfig(t *testing.T) {
	// when
	kubeConfig := composeKubeConfigFromData([]byte("token"), "http://over.the.rainbow", "the-namespace", false)

	// then
	context := kubeConfig.Contexts[kubeConfig.CurrentContext]

	assert.Equal(t, "token", kubeConfig.AuthInfos[context.AuthInfo].Token)
	assert.Equal(t, "http://over.the.rainbow", kubeConfig.Clusters[context.Cluster].Server)
	assert.Equal(t, "the-namespace", context.Namespace)
	assert.False(t, kubeConfig.Clusters[context.Cluster].InsecureSkipTLSVerify)
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

func newToolchainCluster(name, tcNs string, apiEndpoint string, status toolchainv1alpha1.ToolchainClusterStatus) (*toolchainv1alpha1.ToolchainCluster, *corev1.Secret) {
	toolchainCluster, secret := test.NewToolchainClusterWithEndpoint(name, tcNs, "secret", apiEndpoint, status, map[string]string{"namespace": "test-namespace"})
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
