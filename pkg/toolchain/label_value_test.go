package toolchain_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/toolchain"

	"gotest.tools/assert"
)

func TestLabelValue(t *testing.T) {
	t.Run("string to valid value", func(t *testing.T) {
		value := toolchain.ToValidValue("johndoe")
		assert.Equal(t, value, "johndoe")

		value = toolchain.ToValidValue("johndoe-at-test-dot-com")
		assert.Equal(t, value, "johndoe-at-test-dot-com")

		value = toolchain.ToValidValue("johndoe@test.com")
		assert.Equal(t, value, "johndoe-at-test-com")

		value = toolchain.ToValidValue("john.jane.doe@test.com")
		assert.Equal(t, value, "john-jane-doe-at-test-com")
	})
}
