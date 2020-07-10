package toolchaincluster

import (
	"testing"

	"github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/codeready-toolchain/toolchain-common/pkg/test/verify"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestAddToolchainClusterAsMember(t *testing.T) {
	// given & then
	verify.AddToolchainClusterAsMember(t, func(toolchainCluster *v1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// given
		controller, req := prepareReconcile(toolchainCluster, cl, service)

		// when
		_, err := controller.Reconcile(req)
		return err
	})

}

func TestAddToolchainClusterAsHost(t *testing.T) {
	// given & then
	verify.AddToolchainClusterAsHost(t, func(toolchainCluster *v1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// given
		controller, req := prepareReconcile(toolchainCluster, cl, service)

		// when
		_, err := controller.Reconcile(req)
		return err
	})
}

func TestAddToolchainClusterFailsBecauseOfMissingSecret(t *testing.T) {
	// given & then
	verify.AddToolchainClusterFailsBecauseOfMissingSecret(t, func(toolchainCluster *v1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// given
		controller, req := prepareReconcile(toolchainCluster, cl, service)

		// when
		_, err := controller.Reconcile(req)
		return err
	})
}

func TestAddToolchainClusterFailsBecauseOfEmptySecret(t *testing.T) {
	// given & then
	verify.AddToolchainClusterFailsBecauseOfEmptySecret(t, func(toolchainCluster *v1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// given
		controller, req := prepareReconcile(toolchainCluster, cl, service)

		// when
		_, err := controller.Reconcile(req)
		return err
	})
}

func TestUpdateToolchainCluster(t *testing.T) {
	// given & then
	verify.UpdateToolchainCluster(t, func(toolchainCluster *v1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// given
		controller, req := prepareReconcile(toolchainCluster, cl, service)

		// when
		_, err := controller.Reconcile(req)
		return err
	})
}

func TestDeleteToolchainCluster(t *testing.T) {
	// given & then
	verify.DeleteToolchainCluster(t, func(toolchainCluster *v1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) error {
		// given
		controller, req := prepareReconcile(toolchainCluster, cl, service)

		// when
		_, err := controller.Reconcile(req)
		return err
	})
}

func prepareReconcile(toolchainCluster *v1alpha1.ToolchainCluster, cl *test.FakeClient, service cluster.ToolchainClusterService) (ReconcileToolchainCluster, reconcile.Request) {
	controller := ReconcileToolchainCluster{
		client:              cl,
		scheme:              scheme.Scheme,
		clusterCacheService: service,
	}
	req := reconcile.Request{
		NamespacedName: test.NamespacedName(toolchainCluster.Namespace, toolchainCluster.Name),
	}
	return controller, req
}
