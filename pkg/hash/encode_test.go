package hash_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/hash"

	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	// given
	value := "random_content"
	// when
	result := hash.Encode([]byte(value))
	// then
	assert.Equal(t, "25ec617dda1a9ac8f4a2dc346adee4dd", result) // see `echo -n "random_content" | md5`
}

func TestEncodeString(t *testing.T) {
	// given
	value := "random_content"
	// when
	result := hash.EncodeString(value)
	// then
	assert.Equal(t, "25ec617dda1a9ac8f4a2dc346adee4dd", result) // see `echo -n "random_content" | md5`
}
