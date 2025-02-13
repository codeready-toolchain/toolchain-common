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
		Status: toolchainv1alpha1.NSTemplateTierStatus{
			Revisions: map[string]string{
				"base1ns-dev-aeb78eb-aeb78eb":              "base1ns-dev-aeb78eb-aeb78eb",
				"base1ns-clusterresources-e0e1f34-e0e1f34": "base1ns-clusterresources-e0e1f34-e0e1f34",
				"base1ns-admin-123456abc":                  "base1ns-admin-123456abc",
			},
		},
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
