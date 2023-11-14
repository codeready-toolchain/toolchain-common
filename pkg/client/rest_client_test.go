package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	restclient "github.com/codeready-toolchain/toolchain-common/pkg/client"
	clienttest "github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestCreateTokenRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
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
		token, err := restclient.CreateTokenRequest(context.TODO(), cl, types.NamespacedName{
			Namespace: "jane-env",
			Name:      "jane",
		}, 1)

		// then
		require.NoError(t, err)
		assert.Equal(t, "token-secret-for-jane", token) // `token-secret-for-jane` is the answered mock by Gock in `clienttest.SetupGockForServiceAccounts(...)`
	})
	t.Run("failure", func(t *testing.T) {
		t.Run("empty token is returned", func(t *testing.T) {
			// given
			// the api server returns an error an a nil token request
			const apiEndpoint = "https://api.example.com"
			// setting an invalid response body so that client go library will return an empty token string
			invalidResponseObject, err := json.Marshal(nil)
			require.NoError(t, err)
			clienttest.SetupGockWithCleanup(t, apiEndpoint, "/api/v1/namespaces/jane-env/serviceaccounts/jane/token", string(invalidResponseObject), http.StatusOK)
			cl, err := clienttest.NewRESTClient("secret_token", apiEndpoint)
			cl.Client.Transport = gock.DefaultTransport // make sure that the underlying client's request are intercepted by Gock

			// when
			require.NoError(t, err)
			token, err := restclient.CreateTokenRequest(context.TODO(), cl, types.NamespacedName{
				Namespace: "jane-env",
				Name:      "jane",
			}, 1)

			// then
			require.Error(t, err)                                                    // an error should be returned
			assert.Equal(t, "unable to create token, got empty string", err.Error()) // error message should match expected one
			assert.Equal(t, "", token)                                               // token should be empty
		})
	})
}
