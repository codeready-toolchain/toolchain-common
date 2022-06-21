package socialevent_test

import (
	"regexp"
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/socialevent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSocialEventName(t *testing.T) {

	t.Run("single code", func(t *testing.T) {
		// when
		code := socialevent.NewName()
		// then
		match, err := regexp.Match("^[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}$", []byte(code))
		require.NoError(t, err)
		assert.True(t, match, "code '%s' did not match the expected format", code)
	})

	t.Run("multiple codes", func(t *testing.T) {
		// generate a bucket of activation codes and verifies that the fothere is no collision
		codes := make(map[string]bool, 10000)
		for i := 0; i < 1000; i++ {
			code := socialevent.NewName()
			_, found := codes[code]
			require.False(t, found)
			codes[code] = true
		}
	})
}
