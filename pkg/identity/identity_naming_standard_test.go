package identity_test

import (
	"github.com/codeready-toolchain/toolchain-common/pkg/identity"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIdentityNamingStandard(t *testing.T) {

	t.Run("Check plain identity name ok", func(t *testing.T) {
		require.Equal(t, "rhd:john", identity.NewIdentityNamingStandard("john", "rhd").IdentityName())
	})

	t.Run("Check identity name with non-standard chars ok", func(t *testing.T) {
		require.Equal(t, "rhd:b64:am9oblxi", identity.NewIdentityNamingStandard("john\\b", "rhd").IdentityName())
	})
}