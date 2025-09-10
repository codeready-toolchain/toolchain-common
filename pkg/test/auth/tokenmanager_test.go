package auth

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	jose "gopkg.in/go-jose/go-jose.v2"
	"gotest.tools/assert"
)

func TestTokenManagerKeys(t *testing.T) {
	t.Run("create keys", func(t *testing.T) {
		tokenManager := NewTokenManager()
		kid0 := uuid.NewString()
		key0, err := tokenManager.AddPrivateKey(kid0)
		require.NoError(t, err)
		require.NotNil(t, key0)
		kid1 := uuid.NewString()
		key1, err := tokenManager.AddPrivateKey(kid1)
		require.NoError(t, err)
		require.NotNil(t, key1)
		// check key equality by comparing the modulus
		require.NotEqual(t, key0.N, key1.N)
	})

	t.Run("remove keys", func(t *testing.T) {
		tokenManager := NewTokenManager()
		kid0 := uuid.NewString()
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
		kid0 := uuid.NewString()
		key0, err := tokenManager.AddPrivateKey(kid0)
		require.NoError(t, err)
		require.NotNil(t, key0)
		kid1 := uuid.NewString()
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
	kid0 := uuid.NewString()
	key0, err := tokenManager.AddPrivateKey(kid0)
	require.NoError(t, err)
	require.NotNil(t, key0)

	t.Run("create token", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0)
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, identity0.ID.String(), claims.Subject)
	})

	t.Run("create token with email extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithEmailClaim(identity0.Username+"@email.tld"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, identity0.ID.String(), claims.Subject)
	})
	t.Run("create token with iat extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		iatTime := time.Now().Add(-60 * time.Second)
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithIATClaim(iatTime))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, identity0.ID.String(), claims.Subject)
		require.Equal(t, iatTime.Unix(), claims.IssuedAt.Unix())
	})
	t.Run("create token with exp extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		expTime := time.Now().Add(-60 * time.Second)
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithExpClaim(expTime))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.Error(t, err)
		require.EqualError(t, err, "token has invalid claims: token is expired")
		require.False(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, identity0.ID.String(), claims.Subject)
		require.Equal(t, expTime.Unix(), claims.ExpiresAt.Unix())
	})
	t.Run("create token with sub extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithSubClaim("test"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, "test", claims.Subject)
	})
	t.Run("create token with nbf extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		nbfTime := time.Now().Add(60 * time.Second)
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithNotBeforeClaim(nbfTime))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "token is not valid yet")
		require.False(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, identity0.ID.String(), claims.Subject)
		require.Equal(t, nbfTime.Unix(), claims.NotBefore.Unix())
	})
	t.Run("create token with given name extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithGivenNameClaim("jane"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, "jane", claims.GivenName)
	})
	t.Run("create token with family name extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithFamilyNameClaim("doe"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, "doe", claims.FamilyName)
	})
	t.Run("create token with preferred username extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithPreferredUsernameClaim("test-preferred-username"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, "test-preferred-username", claims.PreferredUsername)
	})
	t.Run("create token with company extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithCompanyClaim("red hat"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, "red hat", claims.Company)
	})
	t.Run("create token with near future iat claim to test validation workaround", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		iatTime := time.Now().Add(10 * time.Second)
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithIATClaim(iatTime))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, identity0.ID.String(), claims.Subject)
		require.Equal(t, iatTime.Unix(), claims.IssuedAt.Unix())
	})
	t.Run("create token with original_sub extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithOriginalSubClaim("Jack:1234-FFFF"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, "Jack:1234-FFFF", claims.OriginalSub)
	})
	t.Run("create token with user_id extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithUserIDClaim("123456789"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, "123456789", claims.UserID)
	})
	t.Run("create token with account_id extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithAccountIDClaim("987654321"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, "987654321", claims.AccountID)
	})
	t.Run("create token with account_number extra claim", func(t *testing.T) {
		username := uuid.NewString()
		identity0 := &Identity{
			ID:       uuid.New(),
			Username: username,
		}
		// generate the token
		encodedToken, err := tokenManager.GenerateSignedToken(*identity0, kid0, WithAccountNumberClaim("987654320"))
		require.NoError(t, err)
		// unmarshall it again
		decodedToken, err := jwt.ParseWithClaims(encodedToken, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return &(key0.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, decodedToken.Valid)
		claims, ok := decodedToken.Claims.(*MyClaims)
		require.True(t, ok)
		require.Equal(t, "987654320", claims.AccountNumber)
	})
}

func TestTokenManagerKeyService(t *testing.T) {
	tokenManager := NewTokenManager()
	kid0 := uuid.NewString()
	key0, err := tokenManager.AddPrivateKey(kid0)
	require.NoError(t, err)
	require.NotNil(t, key0)
	kid1 := uuid.NewString()
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
		defer func() {
			_, _ = io.Copy(io.Discard, res.Body)
			defer res.Body.Close()
		}()
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
		require.Len(t, webKey0, 1)
		rsaKey0, ok := webKey0[0].Key.(*rsa.PublicKey)
		require.True(t, ok)
		// check key equality by comparing the modulus
		require.Equal(t, key0.N, rsaKey0.N)

		// check key integrity for key 1
		webKey1 := webKeys.Key(kid1)
		require.NotNil(t, webKey1)
		require.Len(t, webKey1, 1)
		rsaKey1, ok := webKey1[0].Key.(*rsa.PublicKey)
		require.True(t, ok)
		// check key equality by comparing the modulus
		require.Equal(t, key1.N, rsaKey1.N)
	})
}

func TestTokenManagerE2ETestKeys(t *testing.T) {
	identity := NewIdentity()
	emailClaim := WithEmailClaim(uuid.NewString() + "@email.tld")
	token, err := GenerateSignedE2ETestToken(*identity, emailClaim)
	require.NoError(t, err)
	require.NotNil(t, token)

	t.Run("test valid token", func(t *testing.T) {
		publicKeys := GetE2ETestPublicKey()
		require.Len(t, publicKeys, 1)
		publicKey := publicKeys[0]
		parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			kid := token.Header["kid"]
			require.NotNil(t, kid)
			kidStr, ok := kid.(string)
			require.True(t, ok)
			assert.Equal(t, publicKey.KeyID, kidStr)

			return publicKey.Key, nil
		})
		require.NoError(t, err)
		require.True(t, parsedToken.Valid)
	})
}
