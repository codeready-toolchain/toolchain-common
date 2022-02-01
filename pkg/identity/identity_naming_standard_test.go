package identity_test

import (
	"github.com/codeready-toolchain/toolchain-common/pkg/identity"
	v1 "github.com/openshift/api/user/v1"
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

	t.Run("Check apply to identity ok", func(t *testing.T) {
		id := &v1.Identity{}
		identity.NewIdentityNamingStandard("jill", "rhd").ApplyToIdentity(id)
		require.Equal(t, "rhd:jill", id.Name)
		require.Equal(t, "rhd", id.ProviderName)
		require.Equal(t, "jill", id.ProviderUserName)
	})

	t.Run("Check apply to identity ok", func(t *testing.T) {
		id := &v1.Identity{}
		identity.NewIdentityNamingStandard("jill", "rhd").ApplyToIdentity(id)
		require.Equal(t, "rhd:jill", id.Name)
		require.Equal(t, "rhd", id.ProviderName)
		require.Equal(t, "jill", id.ProviderUserName)
	})

	t.Run("Check apply to identity non-standard chars ok", func(t *testing.T) {
		id := &v1.Identity{}
		identity.NewIdentityNamingStandard("jjones:jill@somewhere.com", "rhd").ApplyToIdentity(id)
		require.Equal(t, "rhd:b64:ampvbmVzOmppbGxAc29tZXdoZXJlLmNvbQ", id.Name)
		require.Equal(t, "rhd", id.ProviderName)
		require.Equal(t, "b64:ampvbmVzOmppbGxAc29tZXdoZXJlLmNvbQ", id.ProviderUserName)
	})
}