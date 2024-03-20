package toolchaincluster_resources

import (
	"context"
	"embed"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/template"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//go:embed testdata/cluster-role.yaml
var clusterRoleFS embed.FS

//go:embed testdata/service-account.yaml
var serviceAccountFS embed.FS

func TestToolchainClusterResources(t *testing.T) {
	// given
	// we assume there is already a service account generated in the member operator namespaces
	sa := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-sa",
			Namespace: test.MemberOperatorNs,
		},
	}
	cl := test.NewFakeClient(t, sa)

	t.Run("controller should create service account resource", func(t *testing.T) {
		// then
		controller, req := prepareReconcile(sa, cl, &serviceAccountFS)

		// when
		_, err := controller.Reconcile(context.TODO(), req)
		require.NoError(t, err)
		checkExpectedServiceAccountResources(t, cl)
	})

	t.Run("controller should create cluster role resource", func(t *testing.T) {
		// then
		controller, req := prepareReconcile(sa, cl, &clusterRoleFS)

		// when
		_, err := controller.Reconcile(context.TODO(), req)
		require.NoError(t, err)
		cr := &rbac.ClusterRole{}
		err = cl.Get(context.TODO(), types.NamespacedName{
			Name: "member-toolchaincluster-cr",
		}, cr)
		require.NoError(t, err)
		require.Equal(t, ResourceControllerLabelValue, cr.Labels[toolchainv1alpha1.ProviderLabelKey])
	})

	t.Run("controller should return error when not templates are configured", func(t *testing.T) {
		// then
		controller, req := prepareReconcile(sa, cl, nil) // no templates are passed to the controller initialization

		// when
		_, err := controller.Reconcile(context.TODO(), req)
		require.Error(t, err)
	})
}

func checkExpectedServiceAccountResources(t *testing.T, cl *test.FakeClient) {
	expectedTypes := []client.Object{
		&v1.ServiceAccount{},
		&rbac.Role{},
		&rbac.RoleBinding{},
	}

	for _, resourceType := range expectedTypes {
		resource := resourceType
		err := cl.Get(context.TODO(), types.NamespacedName{
			Namespace: test.MemberOperatorNs,
			Name:      "toolchaincluster-host",
		}, resource)
		require.NoError(t, err)
		require.Equal(t, ResourceControllerLabelValue, resource.GetLabels()[toolchainv1alpha1.ProviderLabelKey])
	}
}

func prepareReconcile(sa *v1.ServiceAccount, cl *test.FakeClient, templates *embed.FS) (Reconciler, reconcile.Request) {
	if templates == nil {
		return emptyReconciler(cl)
	}
	templateObjects, err := template.LoadObjectsFromEmbedFS(templates, &template.Variables{Namespace: sa.Namespace})
	if err != nil {
		return emptyReconciler(cl)
	}
	controller := Reconciler{
		client:          cl,
		scheme:          scheme.Scheme,
		templates:       templates,
		templateObjects: templateObjects,
	}
	req := reconcile.Request{
		NamespacedName: test.NamespacedName(sa.Namespace, sa.Name),
	}
	return controller, req
}

func emptyReconciler(cl *test.FakeClient) (Reconciler, reconcile.Request) {
	return Reconciler{
		client:          cl,
		scheme:          scheme.Scheme,
		templates:       nil,
		templateObjects: nil,
	}, reconcile.Request{}
}
