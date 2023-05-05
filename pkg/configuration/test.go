package configuration

import (
	"context"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	commonclient "github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	testconfig "github.com/codeready-toolchain/toolchain-common/pkg/test/config"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
func UpdateToolchainConfigObjWithReset(t *testing.T, cl commonclient.Client, options ...testconfig.ToolchainConfigOption) *toolchainv1alpha1.ToolchainConfig {
	currentConfig := &toolchainv1alpha1.ToolchainConfig{}
	err := cl.Get(context.TODO(), types.NamespacedName{Namespace: test.HostOperatorNs, Name: "config"}, currentConfig)
	require.NoError(t, err)
	t.Cleanup(ResetCache)
	return testconfig.ModifyToolchainConfigObj(t, cl, options...)
}

func NewMemberOperatorConfigWithReset(t *testing.T, options ...testconfig.MemberOperatorConfigOption) *toolchainv1alpha1.MemberOperatorConfig {
	t.Cleanup(ResetCache)
	return testconfig.NewMemberOperatorConfigObj(options...)
}

func UpdateMemberOperatorConfigWithReset(t *testing.T, cl runtimeclient.Client, options ...testconfig.MemberOperatorConfigOption) *toolchainv1alpha1.MemberOperatorConfig {
	currentConfig := &toolchainv1alpha1.MemberOperatorConfig{}
	err := cl.Get(context.TODO(), types.NamespacedName{Namespace: test.MemberOperatorNs, Name: "config"}, currentConfig)
	require.NoError(t, err)
	t.Cleanup(ResetCache)
	return testconfig.ModifyMemberOperatorConfigObj(currentConfig, options...)
}
