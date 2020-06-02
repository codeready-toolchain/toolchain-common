package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewToolchainObject(t *testing.T) {
	// given
	rb := newRoleBinding("rb-test")

	// when
	toolchainObject, err := NewToolchainObject(rb)

	// then
	require.NoError(t, err)
	verifyRoleBining(t, rb, toolchainObject)
}

func TestNewToolchainObjectWithNilObject(t *testing.T) {
	// when
	toolchainObject, err := NewToolchainObject(nil)

	// then
	require.Error(t, err)
	assert.Equal(t, "the provided object is nil, so the constructor cannot create an instance of ToolchainObject", err.Error())
	assert.Nil(t, toolchainObject)
}

func TestToolchainObjectHasSameFunctions(t *testing.T) {
	// given
	roleBindingTest, err := NewToolchainObject(newRoleBinding("rb-test"))
	require.NoError(t, err)
	roleBindingTest2, err := NewToolchainObject(newRoleBinding("rb-test"))
	require.NoError(t, err)
	roleTest, err := NewToolchainObject(newRole("rb-test"))
	require.NoError(t, err)
	roleBindingSecond, err := NewToolchainObject(newRoleBinding("second"))
	require.NoError(t, err)

	t.Run("don't have the same name", func(t *testing.T) {
		// when
		same := roleBindingTest.HasSameName(roleBindingSecond)

		// then
		assert.False(t, same)
	})

	t.Run("have the same name", func(t *testing.T) {
		// when
		same := roleBindingTest.HasSameName(roleTest)

		// then
		assert.True(t, same)
	})

	t.Run("don't have the same GVK", func(t *testing.T) {
		// when
		same := roleBindingTest.HasSameGvk(roleTest)

		// then
		assert.False(t, same)
	})

	t.Run("have the same name", func(t *testing.T) {
		// when
		same := roleBindingTest.HasSameGvk(roleBindingSecond)

		// then
		assert.True(t, same)
	})

	t.Run("don't have either GVK or name the same", func(t *testing.T) {
		// when
		same := roleBindingTest.HasSameGvkAndName(roleTest)

		// then
		assert.False(t, same)
	})

	t.Run("don't have either GVK or name the same", func(t *testing.T) {
		// when
		same := roleBindingTest.HasSameGvkAndName(roleBindingSecond)

		// then
		assert.False(t, same)
	})

	t.Run("have both GVK and name the same name", func(t *testing.T) {
		// when
		same := roleBindingTest.HasSameGvk(roleBindingTest2)

		// then
		assert.True(t, same)
	})
}

func TestNewComparableToolchainObject(t *testing.T) {
	// given
	rb := newRoleBinding("rb-test")

	t.Run("are not the same", func(t *testing.T) {
		// given
		var notSame CompareToolchainObjects = func(firstObject, secondObject ToolchainObject) (bool, error) {
			return false, nil
		}

		// when
		toolchainObject, err := NewComparableToolchainObject(rb, notSame)
		require.NoError(t, err)
		isSame, err := toolchainObject.IsSame(toolchainObject)

		// then
		require.NoError(t, err)
		assert.False(t, isSame)
		verifyRoleBining(t, rb, toolchainObject)
	})

	t.Run("are the same", func(t *testing.T) {
		// given
		var same CompareToolchainObjects = func(firstObject, secondObject ToolchainObject) (bool, error) {
			return true, nil
		}

		// when
		toolchainObject, err := NewComparableToolchainObject(rb, same)
		require.NoError(t, err)
		isSame, err := toolchainObject.IsSame(toolchainObject)

		// then
		require.NoError(t, err)
		assert.True(t, isSame)
		verifyRoleBining(t, rb, toolchainObject)
	})

	t.Run("comparison returns an error", func(t *testing.T) {
		// given
		var withError CompareToolchainObjects = func(firstObject, secondObject ToolchainObject) (bool, error) {
			return false, fmt.Errorf("some error")
		}

		// when
		toolchainObject, err := NewComparableToolchainObject(rb, withError)
		require.NoError(t, err)
		isSame, err := toolchainObject.IsSame(toolchainObject)

		// then
		require.Error(t, err)
		assert.False(t, isSame)
		verifyRoleBining(t, rb, toolchainObject)
	})
}

func verifyRoleBining(t *testing.T, rb *rbacv1.RoleBinding, toolchainObject ToolchainObject) {
	assert.Equal(t, "rb-test", toolchainObject.GetName())
	assert.Equal(t, "namespace-test", toolchainObject.GetNamespace())
	assert.Len(t, toolchainObject.GetLabels(), 2)
	assert.Equal(t, "first-value", toolchainObject.GetLabels()["firstlabel"])
	assert.Equal(t, "second-value", toolchainObject.GetLabels()["secondlabel"])
	assert.Equal(t, rbacv1.SchemeGroupVersion.WithKind("RoleBinding"), toolchainObject.GetGvk())
	assert.Equal(t, rb, toolchainObject.GetRuntimeObject())
}

func newRole(name string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "namespace-test",
			Labels: map[string]string{
				"firstlabel":  "first-value",
				"secondlabel": "second-value",
			},
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		}},
	}
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
