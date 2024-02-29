package assertions

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
// Note that this method accepts multiple predicates and reports any failures in them using
// the Explain function.
func AssertThat(t *testing.T, object client.Object, predicates ...Predicate[client.Object]) {
	t.Helper()
	message := assertThat(object, predicates...)
	if message != "" {
		assert.Fail(t, "some predicates failed to match", message)
	}
}

// assertThat contains the actual logic of the AssertThat function. This is separated out into
// its own testable function because we cannot cannot capture the result of assert.Fail() in
// another test.
func assertThat(object client.Object, predicates ...Predicate[client.Object]) string {
	results := make([]bool, len(predicates))
	failure := false
	for i, p := range predicates {
		res := p.Matches(object)
		failure = failure || !res
		results[i] = res
	}
	if failure {
		// compose the message
		sb := strings.Builder{}
		sb.WriteString("failed predicates report:")
		for i, p := range predicates {
			if !results[i] {
				sb.WriteRune('\n')
				sb.WriteString(Explain(p, object))
			}
		}
		return sb.String()
	}
	return ""
}

// Explain produces a textual explanation for why the provided predicate didn't match. The explanation
// contains the type name of the predicate, the type of the object and, if the predicate implements
// PredicateMatchFixer interface, a diff between what the object looks like and should have looked like
// to match the predicate. This is best used for logging the explanation of test failures in the end to
// end tests.
//
// The lines beginning with "-" are what was expected, "+" marks the actual values.
//
// Note that this function doesn't actually check if the predicate matches the object so it can produce
// slightly misleading output if called with a predicate that matches given object.
func Explain[T client.Object](predicate Predicate[client.Object], actual T) string {
	// this is used for reporting the type of the predicate
	var reportedPredicateType reflect.Type

	// we want the Is() and Has() to be "transparent" and actually report the type of the
	// inner predicate. Because "cast" (the type that underlies Is() and Has()) is generic,
	// we need to employ a little bit of reflection trickery to get at its inner predicate.
	//
	// If it weren't generic, we could simply use a checked cast. But in case of generic
	// types, the checked cast requires us to specify the generic type. But we don't know
	// that here, hence the pain.
	predVal := reflect.ValueOf(predicate)
	if predVal.Kind() == reflect.Pointer {
		predVal = predVal.Elem()
	}
	typName := predVal.Type().Name()
	if strings.HasPrefix(typName, "cast[") {
		// Interestingly, predVal.FieldByName("Inner").Type() returns the type of the field
		// not the type of the value. So we need to get the actual value using .Interface()
		// and get the type of that. Also notice, that in order to be able to call .Interface()
		// on a field, it needs to be public. In code, we could access cast.inner because
		// we're in the same package, but not with reflection. Go go...
		reportedPredicateType = reflect.TypeOf(predVal.FieldByName("Inner").Interface())
	} else {
		reportedPredicateType = reflect.TypeOf(predicate)
	}
	if reportedPredicateType.Kind() == reflect.Pointer {
		reportedPredicateType = reportedPredicateType.Elem()
	}

	prefix := fmt.Sprintf("predicate '%s' didn't match the object", reportedPredicateType.String())
	fix, ok := predicate.(PredicateMatchFixer[client.Object])
	if !ok {
		return prefix
	}

	expected := fix.FixToMatch(actual.DeepCopyObject().(client.Object))
	diff := cmp.Diff(expected, actual)

	return fmt.Sprintf("%s because of the following differences (- indicates the expected values, + the actual values):\n%s", prefix, diff)
}
