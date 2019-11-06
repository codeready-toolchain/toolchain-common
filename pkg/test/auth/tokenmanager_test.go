package auth

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	jose "gopkg.in/square/go-jose.v2"
)

func TestTokenManagerKeys(t *testing.T) {
	t.Run("create keys", func(t *testing.T) {
		tokenManager := NewTokenManager()
		kid0 := uuid.NewV4().String()
		key0, err := tokenManager.AddPrivateKey(kid0)
		require.NoError(t, err)
		require.NotNil(t, key0)
		kid1 := uuid.NewV4().String()
		key1, err := tokenManager.AddPrivateKey(kid1)
		require.NoError(t, err)
		require.NotNil(t, key1)
		// check key equality by comparing the modulus
		require.NotEqual(t, key0.N, key1.N)
	})

	t.Run("remove keys", func(t *testing.T) {
		tokenManager := NewTokenManager()
		kid0 := uuid.NewV4().String()
		key0, err := tokenManager.AddPrivateKey(kid0)
		require.NoError(t, err)
		require.NotNil(t, key0)
		key0, err = tokenManager.AddPrivateKey(kid0)
		require.NotNil(t, key0)
		require.NoError(t, err)
		key0Retrieved, err := tokenManager.Key(kid0)
		require.NotNil(t, key0Retrieved)
		require.NoError(t, err)
		tokenManager.RemovePrivateKey(kid0)
		_, err = tokenManager.Key(kid0)
		require.Error(t, err)
		require.Equal(t, "given kid does not exist", err.Error())
	})

	t.Run("get key", func(t *testing.T) {
		tokenManager := NewTokenManager()
		kid0 := uuid.NewV4().String()
		key0, err := tokenManager.AddPrivateKey(kid0)
		require.NoError(t, err)
		require.NotNil(t, key0)
		kid1 := uuid.NewV4().String()
		key1, err := tokenManager.AddPrivateKey(kid1)
		require.NoError(t, err)
		require.NotNil(t, key1)
		key0Retrieved, err := tokenManager.Key(kid0)
		require.NoError(t, err)
		require.NotNil(t, key0Retrieved)
		// check key equality by comparing the modulus
		require.Equal(t, key0.N, key0Retrieved.N)
		key1Retrieved, err := tokenManager.Key(kid1)
		require.NoError(t, err)
		require.NotNil(t, key1Retrieved)
		// check key equality by comparing the modulus
		require.Equal(t, key1.N, key1Retrieved.N)
	})
}

func TestTokenManagerTokens(t *testing.T) {
	tokenManager := NewTokenManager()
	kid0 := uuid.NewV4().String()
	key0, err := tokenManager.AddPrivateKey(kid0)
	require.NoError(t, err)
	require.NotNil(t, key0)

	t.Run("create token", func(t *testing.T) {
		username := uuid.NewV4().String()
		identity0 := &Identity{
			ID:       uuid.NewV4(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0)
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*jwt.StandardClaims)
		require.True(t, ok)
		require.Equal(t, identity0.ID.String(), claims.Subject)
	})

	t.Run("create token with email extra claim", func(t *testing.T) {
		username := uuid.NewV4().String()
		identity0 := &Identity{
			ID:       uuid.NewV4(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithEmailClaim(identity0.Username+"@email.tld"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*jwt.StandardClaims)
		require.True(t, ok)
		require.Equal(t, identity0.ID.String(), claims.Subject)
	})
}

func TestTokenManagerKeyService(t *testing.T) {
	tokenManager := NewTokenManager()
	kid0 := uuid.NewV4().String()
	key0, err := tokenManager.AddPrivateKey(kid0)
	require.NoError(t, err)
	require.NotNil(t, key0)
	kid1 := uuid.NewV4().String()
	key1, err := tokenManager.AddPrivateKey(kid1)
	require.NoError(t, err)
	require.NotNil(t, key1)

	t.Run("key fetching", func(t *testing.T) {
		ks := tokenManager.NewKeyServer()
		defer ks.Close()
		keysEndpointURL := ks.URL
		httpClient := http.DefaultClient
		req, err := http.NewRequest("GET", keysEndpointURL, nil)
		require.NoError(t, err)
		res, err := httpClient.Do(req)
		require.NoError(t, err)
		// read and parse response body
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(res.Body)
		require.NoError(t, err)
		bodyBytes := buf.Bytes()

		// if status code was not OK, bail out
		require.Equal(t, http.StatusOK, res.StatusCode)

		// unmarshal the keys
		// note: we're intentionally using jose here, not jwx to have two
		// different jwt implementations interact and to not miss implementation
		// or standards issues in the jose library.
		webKeys := &jose.JSONWebKeySet{}
		err = json.Unmarshal(bodyBytes, &webKeys)
		require.NoError(t, err)

		// check key integrity for key 0
		webKey0 := webKeys.Key(kid0)
		require.NotNil(t, webKey0)
		require.Equal(t, 1, len(webKey0))
		rsaKey0, ok := webKey0[0].Key.(*rsa.PublicKey)
		require.True(t, ok)
		// check key equality by comparing the modulus
		require.Equal(t, key0.N, rsaKey0.N)

		// check key integrity for key 1
		webKey1 := webKeys.Key(kid1)
		require.NotNil(t, webKey1)
		require.Equal(t, 1, len(webKey1))
		rsaKey1, ok := webKey1[0].Key.(*rsa.PublicKey)
		require.True(t, ok)
		// check key equality by comparing the modulus
		require.Equal(t, key1.N, rsaKey1.N)
	})
}

func TestTokenManagerE2ETestKeys(t *testing.T) {
	t.Run("test kid", func(t *testing.T) {
		kid := GetE2ETestKeysKid()
		require.Equal(t, "nBVBNiFNxSiX7Znyg4lUx89HQkV2gtJp11zTP6qLg-4", kid)
	})

	t.Run("test rsa public key", func(t *testing.T) {
		publicKey := GetE2ETestPublicKey()
		require.NotNil(t, publicKey)
		require.Equal(t, int(65537), publicKey.E)
		bigIntStr := "17585685423138064221515344850010465141829225653647610055671508033471331748785541599597686438421660467887485808477138637156594005980448699980569369706826636619853053159648458053935975019090507383175182222936968105869341801194801789872771517673384788419353863256947855057715364342104698219940177752275827068245518017497148531608623801167689103889076415559156613014498165475864496153140202151652823445007808983008418932054472575390902378828235640293107048839125093751837412765673098352746967938742490502621115493521999420406189935566991959573658238305766972345166632085538845942586142044779380741847989387620240625673059"
		require.Equal(t, bigIntStr, publicKey.N.String())
	})
}
