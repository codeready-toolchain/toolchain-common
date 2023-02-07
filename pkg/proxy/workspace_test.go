package proxy_test

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/proxy"

	"github.com/stretchr/testify/require"
)

func TestNewWorkspace(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		// given
		// when
		workspace := proxy.NewWorkspace("test")

		// then
		require.Equal(t, "Workspace", workspace.Kind)
		require.Equal(t, "test", workspace.Name)
		require.Empty(t, workspace.Status.Owner)
		require.Empty(t, workspace.Status.Role)
	})

	t.Run("with options", func(t *testing.T) {
		// given
		// when
		workspace := proxy.NewWorkspace("test", proxy.WithOwner("john"), proxy.WithRole("admin"), proxy.WithNamespaces([]toolchainv1alpha1.SpaceNamespace{
			{
				Name: "john-tenant",
				Type: "default",
			},
		}))

		// then
		require.Equal(t, "Workspace", workspace.Kind)
		require.Equal(t, "test", workspace.Name)
		require.Equal(t, "john", workspace.Status.Owner)
		require.Equal(t, "admin", workspace.Status.Role)
		require.Len(t, workspace.Status.Namespaces, 1)
		require.Equal(t, "john-tenant", workspace.Status.Namespaces[0].Name)
		require.Equal(t, "default", workspace.Status.Namespaces[0].Type)
	})
}
