package toolchainconfig

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

func NewToolchainConfigWithReset(t *testing.T, options ...testconfig.ToolchainConfigOption) *toolchainv1alpha1.ToolchainConfig {
	t.Cleanup(Reset)
	return testconfig.NewToolchainConfig(t, options...)
}

func UpdateToolchainConfigWithReset(t *testing.T, cl client.Client, options ...testconfig.ToolchainConfigOption) *toolchainv1alpha1.ToolchainConfig {
	currentConfig := &toolchainv1alpha1.ToolchainConfig{}
	err := cl.Get(context.TODO(), types.NamespacedName{Namespace: test.HostOperatorNs, Name: "config"}, currentConfig)
	require.NoError(t, err)
	t.Cleanup(Reset)
	return testconfig.ModifyToolchainConfig(t, cl, options...)
}
