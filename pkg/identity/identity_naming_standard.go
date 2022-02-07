package identity

import (
	"encoding/base64"
	"fmt"
	userv1 "github.com/openshift/api/user/v1"
	"regexp"
)

const (
	dns1123Value string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
)

var dns1123ValueRegexp = regexp.MustCompile("^" + dns1123Value + "$")

type IdentityNamingStandard interface {
	ApplyToIdentity(identity *userv1.Identity)
	IdentityName() string
}

type identityNamingStandard struct {
	userID   string
	provider string
}

// NewIdentityNamingStandard creates an identityNamingStandard instance that encapsulates the formal naming standards
// required when creating Identity resources
func NewIdentityNamingStandard(userID, provider string) IdentityNamingStandard {
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

// isIdentityNameCompliant returns true if the specified name is RFC-1123 compliant, otherwise it returns false
func isIdentityNameCompliant(name string) bool {
	if len(name) > 253 {
		return false
	}
	return dns1123ValueRegexp.MatchString(name)
}
