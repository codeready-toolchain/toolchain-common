package identity

import (
	"encoding/base64"
	"fmt"
	userv1 "github.com/openshift/api/user/v1"
	"strings"
)

type NamingStandard interface {
	ApplyToIdentity(identity *userv1.Identity)
	IdentityName() string
}

type identityNamingStandard struct {
	userID   string
	provider string
}

// NewIdentityNamingStandard creates an identityNamingStandard instance that encapsulates the formal naming standards
// required when creating Identity resources
func NewIdentityNamingStandard(userID, provider string) NamingStandard {
	return &identityNamingStandard{
		userID:   userID,
		provider: provider,
	}
}

// ApplyToIdentity sets the resource name, provider name and provider username properties to the correct values
// for the specified identity resource
func (s *identityNamingStandard) ApplyToIdentity(identity *userv1.Identity) {
	identity.Name = s.IdentityName()
	identity.ProviderName = s.provider
	identity.ProviderUserName = s.username()
}

func (s *identityNamingStandard) username() string {
	value := s.userID
	if !isIdentityNameCompliant(value) {
		value = fmt.Sprintf("b64:%s", base64.RawStdEncoding.EncodeToString([]byte(value)))
	}
	return value
}

// IdentityName returns a value that may be used to name an Identity resource
func (s *identityNamingStandard) IdentityName() string {
	return fmt.Sprintf("%s:%s", s.provider, s.username())
}

// isIdentityNameCompliant returns true if the specified name is compliant with the Openshift identity naming standard,
// encapsulated in the code found in the urlEncodeIfNecessary() function, otherwise it returns false.
//
// The code at time of writing can be found at:
//
// https://github.com/openshift/oauth-server/blob/ef385cc3c9d90ee52f6db211ceb751e04ae967f5/pkg/api/types.go#L108
//
// If this should change, then this function must be updated to reflect the changes.
//
func isIdentityNameCompliant(name string) bool {
	return !strings.ContainsAny(name, ":/")
}
