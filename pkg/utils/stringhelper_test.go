package utils_test

import (
	"strings"
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestReallySplit(t *testing.T) {
	assert.Empty(t, utils.SplitCommaSeparatedList(""))
	assert.Equal(t, strings.Split("1,2,3", ","), utils.SplitCommaSeparatedList("1,2,3"))
	assert.Equal(t, strings.Split("1", ","), utils.SplitCommaSeparatedList("1"))
}
