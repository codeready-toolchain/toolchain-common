package hash_test

import (
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/hash"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var Status = toolchainv1alpha1.NSTemplateTierStatus{
	Revisions: map[string]string{
		"base1ns-dev-aeb78eb-aeb78eb":              "base1ns-dev-aeb78eb-aeb78eb",
		"base1ns-clusterresources-e0e1f34-e0e1f34": "base1ns-clusterresources-e0e1f34-e0e1f34",
		"base1ns-admin-123456abc":                  "base1ns-admin-123456abc",
	},
}

func TestTemplateTierHashLabelKey(t *testing.T) {
	// given
	tierName := "base1ns"
	// when
	k := hash.TemplateTierHashLabelKey(tierName)
	// then
	assert.Equal(t, "toolchain.dev.openshift.com/base1ns-tier-hash", k)
}

func TestComputeHashForNSTemplateTier(t *testing.T) {
	// given
	tier := &toolchainv1alpha1.NSTemplateTier{
		Status: Status,
	}
	// when
	h, err := hash.ComputeHashForNSTemplateTier(tier)
	// then
	require.NoError(t, err)
	// verify hash
	md5hash := md5.New() // nolint:gosec
	_, _ = md5hash.Write([]byte(`{"refs":["base1ns-admin-123456abc","base1ns-clusterresources-e0e1f34-e0e1f34","base1ns-dev-aeb78eb-aeb78eb"]}`))
	expected := hex.EncodeToString(md5hash.Sum(nil))
	assert.Equal(t, expected, h)
}

func TestComputeHashForNSTemplateSetSpec(t *testing.T) {
	// given
	s := toolchainv1alpha1.NSTemplateSetSpec{
		Namespaces: []toolchainv1alpha1.NSTemplateSetNamespace{
			{
				TemplateRef: "base1ns-dev-aeb78eb-aeb78eb",
			},
		},
		ClusterResources: &toolchainv1alpha1.NSTemplateSetClusterResources{
			TemplateRef: "base1ns-clusterresources-e0e1f34-e0e1f34",
		},
		SpaceRoles: []toolchainv1alpha1.NSTemplateSetSpaceRole{
			{
				TemplateRef: "base1ns-admin-123456abc",
			},
		},
	}
	// when
	h, err := hash.ComputeHashForNSTemplateSetSpec(s)
	// then
	require.NoError(t, err)
	// verify hash
	md5hash := md5.New() // nolint:gosec
	_, _ = md5hash.Write([]byte(`{"refs":["base1ns-admin-123456abc","base1ns-clusterresources-e0e1f34-e0e1f34","base1ns-dev-aeb78eb-aeb78eb"]}`))
	expected := hex.EncodeToString(md5hash.Sum(nil))
	assert.Equal(t, expected, h)
}
