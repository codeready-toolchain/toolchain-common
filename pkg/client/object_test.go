package client_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/client"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSortedObjectsWithThreeObjects(t *testing.T) {
	// given
	roleBindingA := newRoleBinding("rb-a")
	roleBindingB := newRoleBinding("rb-b")
	roleBindingNamespaceZ := newRoleBinding("rb-a")
	roleBindingNamespaceZ.Namespace = "namespace-z"

	objects := []runtimeclient.Object{
		roleBindingNamespaceZ,
		roleBindingB,
		roleBindingA,
	}

	// when
	sorted := client.SortObjectsByName(objects)

	// then
	assert.Equal(t, roleBindingA, sorted[0])
	assert.Equal(t, roleBindingB, sorted[1])
	assert.Equal(t, roleBindingNamespaceZ, sorted[2])
}

func TestSortObjectsWhenEmpty(t *testing.T) {
	// when
	sorted := client.SortObjectsByName([]runtimeclient.Object{})

	// then
	assert.Empty(t, sorted)
}

func newRoleBinding(name string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "namespace-test",
			Labels: map[string]string{
				"firstlabel":  "first-value",
				"secondlabel": "second-value",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "Role",
			Name: name,
		},
	}
}
