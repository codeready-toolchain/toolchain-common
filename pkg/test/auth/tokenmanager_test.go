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
		kid := GetE2ETestKid()
		require.Equal(t, "d5693c31-7016-46a4-bbe4-867e6d6a3b3a", kid)
	})

	t.Run("test rsa public key", func(t *testing.T) {
		publicKey := GetE2ETestPublicKey()[0]
		require.NotNil(t, publicKey)
		require.Equal(t, "d5693c31-7016-46a4-bbe4-867e6d6a3b3a", publicKey.KeyID)
		require.Equal(t, int(65537), publicKey.Key.E)
		bigIntStr := "21012781801511383044732859183821217007851293710821114895760198260232575745572557529024769148751664085753979348282860415904823461936756670234992900896172585883572839603850502208610282955261600579721486560601197293882543768559983830222520087477177038541810733627888647207786580943343041930725590581760179973525124655055856323394194878135685796010168280014803674316013445277393348892224743777084314411975807648692031098004605647403763360717917223808648474264424929342927778987767236679382264619474583832779556235912879279744180212616393049238365337349361442557516860490302890872095555342995498516268873312641767641207473"
		require.Equal(t, bigIntStr, publicKey.Key.N.String())
	})
}
