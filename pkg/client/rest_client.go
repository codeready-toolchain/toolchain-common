package client

import (
	"context"
	"fmt"
	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

// CreateTokenRequest creates a TokenRequest for a service account using given expiration in seconds.
// Returns the token string and nil if everything went fine, otherwise an empty string and an error is returned in case something went wrong.
func CreateTokenRequest(ctx context.Context, restClient *rest.RESTClient, namespacedName types.NamespacedName, expirationInSeconds int) (string, error) {
	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: ptr.To(int64(expirationInSeconds)),
		},
	}
	result := &authv1.TokenRequest{}
	if err := restClient.Post().
		AbsPath(fmt.Sprintf("api/v1/namespaces/%s/serviceaccounts/%s/token", namespacedName.Namespace, namespacedName.Name)).
		Body(tokenRequest).
		Do(ctx).
		Into(result); err != nil {
		return "", err
	}

	if len(result.Status.Token) == 0 {
		return "", fmt.Errorf("unable to create token, got empty string")
	}

	// return the token string
	return result.Status.Token, nil
}
