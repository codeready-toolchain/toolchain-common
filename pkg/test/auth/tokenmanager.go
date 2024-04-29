package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/pkg/errors"
)

const (
	bitSize       = 2048
	e2ePrivateKID = "d5693c31-7016-46a4-bbe4-867e6d6a3b3a"
)

// WebKeySet represents a JWK Set object.
type WebKeySet struct {
	Keys []jwk.Key `json:"keys"`
}

// PublicKey represents an RSA public key with a Key ID
type PublicKey struct {
	KeyID string
	Key   *rsa.PublicKey
}

// ExtraClaim a function to set claims in the token to generate
type ExtraClaim func(token *jwt.Token)

// WithPreferredUsernameClaim sets the `preferred username` claim in the token to generate
func WithPreferredUsernameClaim(username string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).PreferredUsername = username
	}
}

// WithEmailClaim sets the `email` claim in the token to generate
func WithEmailClaim(email string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).Email = email
	}
}

// WithCompanyClaim sets the `company` claim in the token to generate
func WithCompanyClaim(company string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).Company = company
	}
}

// WithGivenNameClaim sets the `givenName` claim in the token to generate
func WithGivenNameClaim(givenName string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).GivenName = givenName
	}
}

// WithFamilyNameClaim sets the `familyName` claim in the token to generate
func WithFamilyNameClaim(familyName string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).FamilyName = familyName
	}
}

// WithIATClaim sets the `iat` claim in the token to generate
func WithIATClaim(iat time.Time) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).IssuedAt = &jwt.NumericDate{Time: iat}
	}
}

// WithExpClaim sets the `exp` claim in the token to generate
func WithExpClaim(exp time.Time) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).ExpiresAt = &jwt.NumericDate{Time: exp}
	}
}

// WithSubClaim sets the `sub` claim in the token to generate
func WithSubClaim(sub string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).Subject = sub
	}
}

// WithOriginalSubClaim sets the `original_sub` claim in the token to generate
func WithOriginalSubClaim(originalSub string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).OriginalSub = originalSub
	}
}

// WithNotBeforeClaim sets the `nbf` claim in the token to generate
func WithNotBeforeClaim(nbf time.Time) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).NotBefore = &jwt.NumericDate{Time: nbf}
	}
}

// WithUserIDClaim sets the `user_id` claim in the token to generate
func WithUserIDClaim(userID string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).UserID = userID
	}
}

// WithAccountIDClaim sets the `account_id` claim in the token to generate
func WithAccountIDClaim(accountID string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).AccountID = accountID
	}
}

// WithAudClaim sets the `aud` claim in the token to generate
func WithAudClaim(aud []string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).Audience = aud
	}
}

// Identity is a user identity
type Identity struct {
	ID       uuid.UUID
	Username string
	Email    string
}

// NewIdentity returns a new, random identity
func NewIdentity() *Identity {
	username := "testuser-" + uuid.Must(uuid.NewV4()).String()
	return &Identity{
		ID:       uuid.Must(uuid.NewV4()),
		Username: username,
		Email:    username + "@acme.com",
	}
}

// TokenManager represents the test token and key manager.
type TokenManager struct {
	keyMap map[string]*rsa.PrivateKey
}

// NewTokenManager creates a new TokenManager.
func NewTokenManager() *TokenManager {
	tg := &TokenManager{}
	tg.keyMap = make(map[string]*rsa.PrivateKey)
	return tg
}

// AddPrivateKey creates and stores a new key with the given kid.
func (tg *TokenManager) AddPrivateKey(kid string) (*rsa.PrivateKey, error) {
	reader := rand.Reader
	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return nil, err
	}
	tg.keyMap[kid] = key
	return key, nil
}

// addE2ETestPrivateKey gets the private e2e key and stores the key with the e2e kid.
func (tg *TokenManager) addE2ETestPrivateKey() {
	key := getE2ETestPrivateKey()
	tg.keyMap[e2ePrivateKID] = key
}

// RemovePrivateKey removes a key from the list of known keys.
func (tg *TokenManager) RemovePrivateKey(kid string) {
	delete(tg.keyMap, kid)
}

// Key retrieves the key associated with the given kid.
func (tg *TokenManager) Key(kid string) (*rsa.PrivateKey, error) {
	key, ok := tg.keyMap[kid]
	if !ok {
		return nil, errors.New("given kid does not exist")
	}
	return key, nil
}

type MyClaims struct {
	jwt.RegisteredClaims
	IdentityID        string `json:"uuid,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	SessionState      string `json:"session_state,omitempty"`
	Type              string `json:"typ,omitempty"`
	Approved          bool   `json:"approved,omitempty"`
	Name              string `json:"name,omitempty"`
	Company           string `json:"company,omitempty"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	Email             string `json:"email,omitempty"`
	EmailVerified     bool   `json:"email_verified,omitempty"`
	OriginalSub       string `json:"original_sub"`
	UserID            string `json:"user_id"`
	AccountID         string `json:"account_id"`
}

// GenerateToken generates a default token.
func (tg *TokenManager) GenerateToken(identity Identity, kid string, extraClaims ...ExtraClaim) *jwt.Token {
	token := jwt.New(jwt.SigningMethodRS256)

	token.Claims = &MyClaims{RegisteredClaims: jwt.RegisteredClaims{
		ID:        uuid.Must(uuid.NewV4()).String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "codeready-toolchain",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
		NotBefore: jwt.NewNumericDate(time.Time{}),
		Subject:   identity.ID.String(),
	},
		IdentityID:        identity.ID.String(),
		PreferredUsername: identity.Username,
		SessionState:      uuid.Must(uuid.NewV4()).String(),
		Type:              "Bearer",
		Approved:          true,
		Name:              "Test User",
		Company:           "Company Inc.",
		GivenName:         "Test",
		FamilyName:        "User",
		EmailVerified:     true,
	}

	for _, extra := range extraClaims {
		extra(token)
	}

	token.Header["kid"] = kid

	return token
}

// SignToken signs a given token using the given private key.
func (tg *TokenManager) SignToken(token *jwt.Token, kid string) (string, error) {
	key, err := tg.Key(kid)
	if err != nil {
		return "", err
	}
	tokenStr, err := token.SignedString(key)
	if err != nil {
		panic(errors.WithStack(err))
	}
	return tokenStr, nil
}

// GenerateSignedToken generates a JWT user token and signs it using the given private key.
func (tg *TokenManager) GenerateSignedToken(identity Identity, kid string, extraClaims ...ExtraClaim) (string, error) {
	token := tg.GenerateToken(identity, kid, extraClaims...)
	return tg.SignToken(token, kid)
}

func GenerateSignedE2ETestToken(identity Identity, extraClaims ...ExtraClaim) (string, error) {
	tm := NewTokenManager()
	tm.addE2ETestPrivateKey()
	return tm.GenerateSignedToken(identity, e2ePrivateKID, extraClaims...)
}

// NewKeyServer creates and starts a http key server
func (tg *TokenManager) NewKeyServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		keySet := &WebKeySet{}
		for kid, key := range tg.keyMap {
			newKey, err := jwk.New(&key.PublicKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = newKey.Set(jwk.KeyIDKey, kid)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			keySet.Keys = append(keySet.Keys, newKey)
		}
		jsonKeyData, err := json.Marshal(keySet)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, string(jsonKeyData))
	}))
}

// GetE2ETestPublicKey returns the public key and kid used for e2e tests
func GetE2ETestPublicKey() []*PublicKey {
	publicKeys := []*PublicKey{}
	key := &PublicKey{
		KeyID: e2ePrivateKID,
		Key:   &getE2ETestPrivateKey().PublicKey,
	}
	publicKeys = append(publicKeys, key)

	return publicKeys
}

var (
	e2eTestPrivateKey *rsa.PrivateKey
	e2ePKOnce         sync.Once
)

// getE2ETestPrivateKey returns the e2e private key from the PEM.
func getE2ETestPrivateKey() *rsa.PrivateKey {
	e2ePKOnce.Do(func() {
		pk, err := rsa.GenerateKey(rand.Reader, bitSize)
		if err != nil {
			return
		}

		if err := pk.Validate(); err != nil {
			return
		}

		e2eTestPrivateKey = pk
	})
	return e2eTestPrivateKey
}
