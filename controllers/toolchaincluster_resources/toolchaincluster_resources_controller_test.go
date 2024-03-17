package toolchaincluster_resources

import (
	"context"
	"embed"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	templatetest "github.com/codeready-toolchain/toolchain-common/pkg/template"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestToolchainClusterResources(t *testing.T) {
	// given
	status := test.NewClusterStatus(toolchainv1alpha1.ToolchainClusterReady, v1.ConditionTrue)
	toolchainCluster, sec := test.NewToolchainCluster("east", test.MemberOperatorNs, "secret", status, nil)
	toolchainCluster.Spec.CABundle = "ZHVtbXk="
	cl := test.NewFakeClient(t, toolchainCluster, sec)
	service := cluster.NewToolchainClusterServiceWithClient(cl, logf.Log, test.MemberOperatorNs, 3*time.Second, func(config *rest.Config, options client.Options) (client.Client, error) {
		return client.New(config, options)
	})
	defer service.DeleteToolchainCluster("east")

	t.Run("controller should create service account resource", func(t *testing.T) {
		// then
		controller, req := prepareReconcile(toolchainCluster, cl, &templatetest.HostFS)

		// when
		_, err := controller.Reconcile(context.TODO(), req)
		require.NoError(t, err)
		sa := &v1.ServiceAccount{}
		err = cl.Get(context.TODO(), types.NamespacedName{
			Namespace: test.MemberOperatorNs,
			Name:      "toolchaincluster-host",
		}, sa)
		require.NoError(t, err)
		require.Equal(t, toolchainv1alpha1.ProviderLabelValue, sa.Labels[toolchainv1alpha1.ProviderLabelKey])
	})

	t.Run("controller should create cluster role resource", func(t *testing.T) {
		// then
		controller, req := prepareReconcile(toolchainCluster, cl, &templatetest.MemberFS)

		// when
		_, err := controller.Reconcile(context.TODO(), req)
		require.NoError(t, err)
		cr := &rbac.ClusterRole{}
		err = cl.Get(context.TODO(), types.NamespacedName{
			Name: "member-toolchaincluster-cr",
		}, cr)
		require.NoError(t, err)
		require.Equal(t, toolchainv1alpha1.ProviderLabelValue, cr.Labels[toolchainv1alpha1.ProviderLabelKey])
	})

	t.Run("controller should return error when not templates are configured", func(t *testing.T) {
		// then
		controller, req := prepareReconcile(toolchainCluster, cl, nil) // no templates are passed to the controller initialization

		// when
		_, err := controller.Reconcile(context.TODO(), req)
		require.Error(t, err)
	})
}

func prepareReconcile(toolchainCluster *toolchainv1alpha1.ToolchainCluster, cl *test.FakeClient, templates *embed.FS) (Reconciler, reconcile.Request) {
	controller := Reconciler{
		client:    cl,
		scheme:    scheme.Scheme,
		templates: templates,
	}
	req := reconcile.Request{
		NamespacedName: test.NamespacedName(toolchainCluster.Namespace, toolchainCluster.Name),
	}
	return controller, req
}
