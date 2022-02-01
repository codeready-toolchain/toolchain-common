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

	t.Run("Check identity name with excessive length ok", func(t *testing.T) {
		userID := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789" +
			"abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789" +
			"abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789"

		require.Equal(t, "rhd:b64:YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5YWJjZGVmZ2hpamtsbW5vcHFyc3R1" +
			"dnd4eXowMTIzNDU2Nzg5YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMT" +
			"IzNDU2Nzg5YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5" +
			"YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5YWJjZGVmZ2" +
			"hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5", identity.NewIdentityNamingStandard(userID, "rhd").IdentityName())
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
