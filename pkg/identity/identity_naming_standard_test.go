package identity_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/identity"
	v1 "github.com/openshift/api/user/v1"
	"github.com/stretchr/testify/require"
)

func TestIdentityNamingStandard(t *testing.T) {

	t.Run("Check plain identity name ok", func(t *testing.T) {
		require.Equal(t, "rhd:john", identity.NewIdentityNamingStandard("john", "rhd").IdentityName())
	})

	t.Run("Check identity name with non-standard chars encoded ok", func(t *testing.T) {
		require.Equal(t, "rhd:b64:am9obi9i", identity.NewIdentityNamingStandard("john/b", "rhd").IdentityName())

		require.Equal(t, "rhd:b64:amFjazphYmM", identity.NewIdentityNamingStandard("jack:abc", "rhd").IdentityName())
	})

	t.Run("Check apply to identity ok", func(t *testing.T) {
		id := &v1.Identity{}
		identity.NewIdentityNamingStandard("jill", "rhd").ApplyToIdentity(id)
		require.Equal(t, "rhd:jill", id.Name)
		require.Equal(t, "rhd", id.ProviderName)
		require.Equal(t, "jill", id.ProviderUserName)
	})

	t.Run("Check apply to identity ok for userID with minus prefix", func(t *testing.T) {
		id := &v1.Identity{}
		identity.NewIdentityNamingStandard("-194567", "rhd").ApplyToIdentity(id)
		require.Equal(t, "rhd:-194567", id.Name)
		require.Equal(t, "rhd", id.ProviderName)
		require.Equal(t, "-194567", id.ProviderUserName)
	})

	t.Run("Check apply to identity non-standard chars ok", func(t *testing.T) {
		id := &v1.Identity{}
		identity.NewIdentityNamingStandard("jjones:jill@somewhere.com", "rhd").ApplyToIdentity(id)
		require.Equal(t, "rhd:b64:ampvbmVzOmppbGxAc29tZXdoZXJlLmNvbQ", id.Name)
		require.Equal(t, "rhd", id.ProviderName)
		require.Equal(t, "b64:ampvbmVzOmppbGxAc29tZXdoZXJlLmNvbQ", id.ProviderUserName)
	})
}
