package client_test

import (
	"testing"

	restclient "github.com/codeready-toolchain/toolchain-common/pkg/client"
	clienttest "github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestCreateTokenRequest(t *testing.T) {
	// given
	const apiEndpoint = "https://api.example.com"
	clienttest.SetupGockForServiceAccounts(t, apiEndpoint, types.NamespacedName{
		Name:      "jane",
		Namespace: "jane-env",
	})
	cl, err := clienttest.NewRESTClient("secret_token", apiEndpoint)
	cl.Client.Transport = gock.DefaultTransport // make sure that the underlying client's request are intercepted by Gock

	// when
	require.NoError(t, err)
	token, err := restclient.CreateTokenRequest(cl, types.NamespacedName{
		Namespace: "jane-env",
		Name:      "jane",
	}, 1)

	// then
	require.NoError(t, err)
	assert.Equal(t, "token-secret-for-jane", token) // `token-secret-for-jane` is the answered mock by Gock in `clienttest.SetupGockForServiceAccounts(...)`
}
