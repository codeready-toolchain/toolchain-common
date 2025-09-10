package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/pkg/errors"
)

const (
	bitSize = 2048
	//nolint:gosec
	e2ePrivatePEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEApnQLIhfCVZPJKt5D5SCRUhJ/N5aCsRNlnowqMFzhUF7DF5kb
YWoE8YWF6YcLuyfh/NChAVkixd4zOvyOtVuOjFao/1/2HmKlGxeJ4JhlF1PBXMZV
L53aInEaP4A8J5kAghN74P+Uz1ax1/eF8FjV711ETZDiwYUYXvbPaIdb8WvCU7tG
A5v63My+6PrrDia1xgOevOicV/qxKWdb3stFQ52x/hJKHuMbyGTjSJ6tXdnJZ3ND
j04OBLI0Z1uNShHcGPqp9foAX02dGEJvmBorDg7O1egVNGRYEK7DJ8Y0T50EXGpr
gJaSYjYMTL6u2Ds9vLzjircigD+F2ltJdbhSsQIDAQABAoIBADBsB6UWVlFA2b+f
ww6Pp9bBTMLmBQTwSJqT2d4R1vXja0udHar8BY4hMrCZuZ7rXkGGi5/xxzzag/q/
59/4T4Kh3y3TQ6zZM4CrG0/75USg99o+VB+zAvcMAf/BFT7LsqskceAlWavrY3cZ
KZyeqzWj4y/RWzXCuzE9CV82KVgUcccKofwK6ZauwXDke2xRruaOMeJ4mP62xgNp
hVy0W/La5sqrq24EzJ/0hEMJYg+Z0udOzLofl5NqAoPrazgdZg1oVxbGY0sSUEax
kA/nIlUskiNTgCYrRAeWrI1p6L0LtKMQ+KMs5ek5lI3k2K6EViHXO5kelOKeIas0
hVo0tfECgYEA2NeYtkPIZDzGonu60/52FpJyLzoW9mxc8UBa9/p/CgMC/UzdyxbL
ys4Tw/BuXxwPx0shAI/txlfqd3Dl9z3HF+e84VOIph3VqYFh9cBkZQI9z55pP5kt
o8UW1SWUA799QTIZRhdFrPspaPISiWXgGAiHfaOy6SMM/ghTU22+Dm0CgYEAxIME
lycBt7dsfvbb41OsVeH61mYeC7ZB6FNLhF7X2CqH9ybhMGqUnYvN+/EHMElWR/ky
xe68Hcsvq3sSmEv1SHjAk6WottjpdwwCXvDKWu3LEjR6o3i2VRTCL1jJD9OlcJnk
tSdI2gp/rTQrcm/ANY9KcmYfAyq/xe7DkOkUWtUCgYEAuAUXKy6Q5EgThhacsYXU
L0mur1eL3yqNIYus559kqllt8wqFevFolz6V1YW4FOzakxW19yUt81Huv9hGwLBj
wmy+hTZ/1AGjrksHmCfiyznAvO5BgWB8M+xxeQd/+kJKiMZ8XlgnoCoxtUch5gpX
x+2NFlmS3nkJcJgeJsIONW0CgYAPW7YGIjROKXW/TofM8oMriyfRjdWXUL1B7RCf
3dG8wUYzGMTMxeerkHuezy2ipnip014WfhwRsAmfu1SutnELIvTaFT5kW/uTJEsj
JGqMRL10RMm48Pw/Fgo/LQ85v27UqBJp3hIhiGSGIueqX/WDuhk1a6nM05B9ZbW/
I5hFqQKBgEktcozzuQL0EcyTJ+wFPSoma4qdAqbYf4sUWC9ebrzVd2/plhVRren7
nmblwgPUKfdPKPe9ckWQOaHAIpNsq5Baxjq2wxFWZOvxH2qWmVmljEeoiTRdTHoF
sMnQfhExyZp/T6uc3rgP0yyOFzSbZrnXpzZ9CZtfqbsfjGKwEbq7
-----END RSA PRIVATE KEY-----
`
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

// WithAccountNumberClaim sets the `account_number` claim in the token to generate
func WithAccountNumberClaim(accountNumber string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(*MyClaims).AccountNumber = accountNumber
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
	username := "testuser-" + uuid.NewString()
	return &Identity{
		ID:       uuid.New(),
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
	AccountNumber     string `json:"account_number,omitempty"`
}

// GenerateToken generates a default token.
func (tg *TokenManager) GenerateToken(identity Identity, kid string, extraClaims ...ExtraClaim) *jwt.Token {
	token := jwt.New(jwt.SigningMethodRS256)

	token.Claims = &MyClaims{RegisteredClaims: jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "codeready-toolchain",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
		NotBefore: jwt.NewNumericDate(time.Time{}),
		Subject:   identity.ID.String(),
	},
		IdentityID:        identity.ID.String(),
		PreferredUsername: identity.Username,
		SessionState:      uuid.NewString(),
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

// getE2ETestPrivateKey returns the e2e private key from the PEM.
func getE2ETestPrivateKey() *rsa.PrivateKey {
	r := strings.NewReader(e2ePrivatePEM)
	pemBytes, err := io.ReadAll(r)
	if err != nil {
		return nil
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil
	}

	return privateKey
}
