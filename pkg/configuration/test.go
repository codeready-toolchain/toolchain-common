package configuration

import (
	"context"
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	testconfig "github.com/codeready-toolchain/toolchain-common/pkg/test/config"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewToolchainConfigObjWithReset creates a ToolchainConfig object and adds the cache Reset to the test cleanup.
// It is located here to prevent import cycles between this package and the test package.
func NewToolchainConfigObjWithReset(t *testing.T, options ...testconfig.ToolchainConfigOption) *toolchainv1alpha1.ToolchainConfig {
	t.Cleanup(ResetCache)
	return testconfig.NewToolchainConfigObj(t, options...)
}

// UpdateToolchainConfigObjWithReset updates the ToolchainConfig resource with the name "config" found using the provided client and updated using the provided options.
// Also adds the cache Reset to the test cleanup.
// It is located here to prevent import cycles between this package and the test package.
func UpdateToolchainConfigObjWithReset(t *testing.T, cl client.Client, options ...testconfig.ToolchainConfigOption) *toolchainv1alpha1.ToolchainConfig {
	currentConfig := &toolchainv1alpha1.ToolchainConfig{}
	err := cl.Get(context.TODO(), types.NamespacedName{Namespace: test.HostOperatorNs, Name: "config"}, currentConfig)
	require.NoError(t, err)
	t.Cleanup(ResetCache)
	return testconfig.ModifyToolchainConfigObj(t, cl, options...)
}
