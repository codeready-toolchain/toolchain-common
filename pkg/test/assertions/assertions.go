package assertions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AssertThat is a helper function that tests that the provided object satisfies given predicate.
// It is a exactly implemented as:
//
//	assert.True(t, predicate.Matches(object))
//
// but provides better readability. Instead of:
//
//	assert.True(t, Is(Named("asdf")).Matches(object))
//
// one can write:
//
//	AssertThat(t, object, Is(Named("asdf")))
//
// Note that it intentionally doesn't accept a slice of predicates so that it is easy
// to spot the failed predicate in the actual tests and the caller doesn't have to guess
// which of the many supplied predicates might have failed.
func AssertThat(t *testing.T, object client.Object, predicate Predicate[client.Object]) {
	assert.True(t, predicate.Matches(object))
}
