package test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// SetupGockForServiceAccounts registers the /namespaces/<namespace>/serviceaccounts/<serviceaccount>/token endpoint with gock.
// so that token request are intercepted by gock.
func SetupGockForServiceAccounts(t *testing.T, apiEndpoint string, sas ...types.NamespacedName) {
	for _, sa := range sas {
		expectedToken := "token-secret-for-" + sa.Name
		resultTokenRequest := &authv1.TokenRequest{
			Status: authv1.TokenRequestStatus{
				Token: expectedToken,
			},
		}
		resultTokenRequestStr, err := json.Marshal(resultTokenRequest)
		require.NoError(t, err)
		path := fmt.Sprintf("api/v1/namespaces/%s/serviceaccounts/%s/token", sa.Namespace, sa.Name)
		t.Logf("mocking access to POST %s/%s", apiEndpoint, path)
		SetupGockWithCleanup(t, apiEndpoint, path, string(resultTokenRequestStr), http.StatusOK)
	}
}

func SetupGockWithCleanup(t *testing.T, apiEndpoint string, path string, body string, statusCode int) *gock.Response {
	request := gock.New(apiEndpoint).
		Post(path).
		Persist().
		Reply(statusCode).
		BodyString(body)
	t.Cleanup(gock.OffAll)
	return request
}

// NewRESTClient returns a new kube api rest client.
func NewRESTClient(token, apiEndpoint string) (*rest.RESTClient, error) {
	config := &rest.Config{
		BearerToken: token,
		Host:        apiEndpoint,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // nolint: gosec
			},
		},
		// These fields need to be set when using the REST client
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &authv1.SchemeGroupVersion,
			NegotiatedSerializer: scheme.Codecs,
		},
	}
	return rest.RESTClientFor(config)
}
