package test

import (
	"strings"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestExplain(t *testing.T) {
	t.Run("with diff", func(t *testing.T) {
		// given
		actual := &corev1.Secret{}
		actual.SetName("actual")

		pred := Has(Name("expected"))

		// when
		expl := Explain(pred, actual)

		// then
		assert.True(t, strings.HasPrefix(expl, "predicate 'test.named' didn't match the object because of the following differences (- indicates the expected values, + the actual values):"))
		assert.Contains(t, expl, "-")
		assert.Contains(t, expl, "\"expected\"")
		assert.Contains(t, expl, "+")
		assert.Contains(t, expl, "\"actual\"")
	})

	t.Run("without diff", func(t *testing.T) {
		// given
		actual := &corev1.Secret{}
		actual.SetName("actual")

		pred := Is[client.Object](&predicateWithoutFixing{})

		// when
		expl := Explain(pred, actual)

		// then
		assert.Equal(t, "predicate 'test.predicateWithoutFixing' didn't match the object", expl)
	})

	t.Run("with a slice", func(t *testing.T) {
		actual := []int{1, 2, 3}
		pred := MockPredicate[[]int]{}
		pred.MatchesFunc = func(v []int) bool {
			return false
		}
		pred.FixToMatchFunc = func(v []int) []int {
			return []int{1, 2}
		}

		expl := Explain(Is[[]int](pred), actual)

		assert.True(t, strings.HasPrefix(expl, "predicate 'test.MockPredicate[[]int]' didn't match the object because of the following"))
	})

	t.Run("with conditions", func(t *testing.T) {
		actual := []toolchainv1alpha1.Condition{
			{
				Type:   toolchainv1alpha1.ConditionType("test"),
				Status: corev1.ConditionFalse,
				Reason: "because",
			},
		}

		pred := ConditionThat(toolchainv1alpha1.ConditionType("test"), HasStatus(corev1.ConditionTrue))

		expl := Explain(Has(pred), actual)

		assert.True(t, strings.HasPrefix(expl, "predicate 'test.conditionsPredicate' didn't match the object because of the following"))
	})
}

func TestAssertThat(t *testing.T) {
	t.Run("positive case", func(t *testing.T) {
		// given
		actual := &corev1.ConfigMap{}
		actual.SetName("actual")
		actual.SetLabels(map[string]string{"k": "v"})

		// when
		message := assertThat(actual, Has(Name("actual")), Has(Labels(map[string]string{"k": "v"})))

		// then
		assert.Empty(t, message)
	})

	t.Run("negative case", func(t *testing.T) {
		// given
		actual := &corev1.ConfigMap{}
		actual.SetName("actual")
		actual.SetLabels(map[string]string{"k": "v"})

		// when
		message := assertThat(actual, Has(Name("expected")), Has(Labels(map[string]string{"k": "another value"})))

		// then
		assert.Contains(t, message, "predicate 'test.named' didn't match the object because of the following differences")
		assert.Contains(t, message, "predicate 'test.hasLabels' didn't match the object because of the following differences")
	})
}

type predicateWithoutFixing struct{}

var _ Predicate[client.Object] = (*predicateWithoutFixing)(nil)

func (*predicateWithoutFixing) Matches(obj client.Object) bool {
	return false
}

type MockPredicate[T any] struct {
	MatchesFunc    func(v T) bool
	FixToMatchFunc func(v T) T
}

func (p MockPredicate[T]) Matches(v T) bool {
	return p.MatchesFunc(v)
}

func (p MockPredicate[T]) FixToMatch(v T) T {
	return p.FixToMatchFunc(v)
}
