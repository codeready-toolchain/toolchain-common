package featuretoggle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatureToggleAnnotationKey(t *testing.T) {
	key := FeatureToggleAnnotationKey("my-cool-feature")
	assert.Equal(t, "toolchain.dev.openshift.com/feature/my-cool-feature", key)
}
