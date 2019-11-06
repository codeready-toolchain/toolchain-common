package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	bitSize = 2048
	e2ePem  = "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEApnQLIhfCVZPJKt5D5SCRUhJ/N5aCsRNlnowqMFzhUF7DF5kb\nYWoE8YWF6YcLuyfh/NChAVkixd4zOvyOtVuOjFao/1/2HmKlGxeJ4JhlF1PBXMZV\nL53aInEaP4A8J5kAghN74P+Uz1ax1/eF8FjV711ETZDiwYUYXvbPaIdb8WvCU7tG\nA5v63My+6PrrDia1xgOevOicV/qxKWdb3stFQ52x/hJKHuMbyGTjSJ6tXdnJZ3ND\nj04OBLI0Z1uNShHcGPqp9foAX02dGEJvmBorDg7O1egVNGRYEK7DJ8Y0T50EXGpr\ngJaSYjYMTL6u2Ds9vLzjircigD+F2ltJdbhSsQIDAQABAoIBADBsB6UWVlFA2b+f\nww6Pp9bBTMLmBQTwSJqT2d4R1vXja0udHar8BY4hMrCZuZ7rXkGGi5/xxzzag/q/\n59/4T4Kh3y3TQ6zZM4CrG0/75USg99o+VB+zAvcMAf/BFT7LsqskceAlWavrY3cZ\nKZyeqzWj4y/RWzXCuzE9CV82KVgUcccKofwK6ZauwXDke2xRruaOMeJ4mP62xgNp\nhVy0W/La5sqrq24EzJ/0hEMJYg+Z0udOzLofl5NqAoPrazgdZg1oVxbGY0sSUEax\nkA/nIlUskiNTgCYrRAeWrI1p6L0LtKMQ+KMs5ek5lI3k2K6EViHXO5kelOKeIas0\nhVo0tfECgYEA2NeYtkPIZDzGonu60/52FpJyLzoW9mxc8UBa9/p/CgMC/UzdyxbL\nys4Tw/BuXxwPx0shAI/txlfqd3Dl9z3HF+e84VOIph3VqYFh9cBkZQI9z55pP5kt\no8UW1SWUA799QTIZRhdFrPspaPISiWXgGAiHfaOy6SMM/ghTU22+Dm0CgYEAxIME\nlycBt7dsfvbb41OsVeH61mYeC7ZB6FNLhF7X2CqH9ybhMGqUnYvN+/EHMElWR/ky\nxe68Hcsvq3sSmEv1SHjAk6WottjpdwwCXvDKWu3LEjR6o3i2VRTCL1jJD9OlcJnk\ntSdI2gp/rTQrcm/ANY9KcmYfAyq/xe7DkOkUWtUCgYEAuAUXKy6Q5EgThhacsYXU\nL0mur1eL3yqNIYus559kqllt8wqFevFolz6V1YW4FOzakxW19yUt81Huv9hGwLBj\nwmy+hTZ/1AGjrksHmCfiyznAvO5BgWB8M+xxeQd/+kJKiMZ8XlgnoCoxtUch5gpX\nx+2NFlmS3nkJcJgeJsIONW0CgYAPW7YGIjROKXW/TofM8oMriyfRjdWXUL1B7RCf\n3dG8wUYzGMTMxeerkHuezy2ipnip014WfhwRsAmfu1SutnELIvTaFT5kW/uTJEsj\nJGqMRL10RMm48Pw/Fgo/LQ85v27UqBJp3hIhiGSGIueqX/WDuhk1a6nM05B9ZbW/\nI5hFqQKBgEktcozzuQL0EcyTJ+wFPSoma4qdAqbYf4sUWC9ebrzVd2/plhVRren7\nnmblwgPUKfdPKPe9ckWQOaHAIpNsq5Baxjq2wxFWZOvxH2qWmVmljEeoiTRdTHoF\nsMnQfhExyZp/T6uc3rgP0yyOFzSbZrnXpzZ9CZtfqbsfjGKwEbq7\n-----END RSA PRIVATE KEY-----\n"
	e2eKid  = "d5693c31-7016-46a4-bbe4-867e6d6a3b3a"
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

// WithEmailClaim sets the `email` claim in the token to generate
func WithEmailClaim(email string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(jwt.MapClaims)["email"] = email
	}
}

// Identity is a user identity
type Identity struct {
	ID       uuid.UUID
	Username string
}

// NewIdentity returns a new, random identity
func NewIdentity() *Identity {
	return &Identity{
		ID:       uuid.NewV4(),
		Username: "testuser-" + uuid.NewV4().String(),
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

// AddE2EPrivateKey creates and stores a the e2e key with the given e2e kid.
func (tg *TokenManager) AddE2EPrivateKey() *rsa.PrivateKey {
	key := getE2ETestPrivateKey()
	tg.keyMap[e2eKid] = key
	return key
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

// GenerateToken generates a default token.
func (tg *TokenManager) GenerateToken(identity Identity, kid string, extraClaims ...ExtraClaim) *jwt.Token {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["uuid"] = identity.ID
	token.Claims.(jwt.MapClaims)["preferred_username"] = identity.Username
	token.Claims.(jwt.MapClaims)["sub"] = identity.ID
	token.Claims.(jwt.MapClaims)["jti"] = uuid.NewV4().String()
	token.Claims.(jwt.MapClaims)["session_state"] = uuid.NewV4().String()
	token.Claims.(jwt.MapClaims)["iat"] = time.Now().Unix()
	token.Claims.(jwt.MapClaims)["exp"] = time.Now().Unix() + 60*60*24*30
	token.Claims.(jwt.MapClaims)["nbf"] = 0
	token.Claims.(jwt.MapClaims)["iss"] = "codeready-toolchain"
	token.Claims.(jwt.MapClaims)["typ"] = "Bearer"
	token.Claims.(jwt.MapClaims)["approved"] = true
	token.Claims.(jwt.MapClaims)["name"] = "Test User"
	token.Claims.(jwt.MapClaims)["company"] = "Company Inc."
	token.Claims.(jwt.MapClaims)["given_name"] = "Test"
	token.Claims.(jwt.MapClaims)["family_name"] = "User"
	token.Claims.(jwt.MapClaims)["email_verified"] = true
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

// NewKeyServer creates and starts an http key server
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
		KeyID: e2eKid,
		Key:   &getE2ETestPrivateKey().PublicKey,
	}

	publicKeys = append(publicKeys, key)

	return publicKeys
}

// GetE2ETestKid returns the e2e kid
func GetE2ETestKid() string {
	return e2eKid
}

func getE2ETestPrivateKey() *rsa.PrivateKey {
	r := strings.NewReader(e2ePem)
	pemBytes, err := ioutil.ReadAll(r)
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
