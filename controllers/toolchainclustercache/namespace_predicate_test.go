package toolchainclustercache

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestNamespacePredicate(t *testing.T) {
	// given
	predicate := namespacePredicate{
		namespace: "matching-namespace",
	}

	tests := map[string]struct {
		namespace string
		expected  bool
	}{
		"with matching namespace": {
			namespace: "matching-namespace",
			expected:  true,
		},

		"without matching namespace": {
			namespace: "non-matching-namespace",
			expected:  false,
		},

		"without any namespace": {
			namespace: "",
			expected:  false,
		},
	}

	for testName, data := range tests {
		t.Run(testName, func(t *testing.T) {
			tc := &toolchainv1alpha1.ToolchainCluster{
				ObjectMeta: v1.ObjectMeta{
					Name:      "tc-name",
					Namespace: data.namespace,
				},
			}
			t.Run("update event", func(t *testing.T) {
				// given
				ev := event.UpdateEvent{
					ObjectNew: tc,
					// we don't care about the old version of the object
					ObjectOld: nil,
				}

				// when
				shouldReconcile := predicate.Update(ev)

				// then
				assert.Equal(t, data.expected, shouldReconcile)
			})

			t.Run("create event", func(t *testing.T) {
				// given
				ev := event.CreateEvent{
					Object: tc,
				}

				// when
				shouldReconcile := predicate.Create(ev)

				// then
				assert.Equal(t, data.expected, shouldReconcile)
			})

			t.Run("generic event", func(t *testing.T) {
				// given
				ev := event.GenericEvent{
					Object: tc,
				}

				// when
				shouldReconcile := predicate.Generic(ev)

				// then
				assert.Equal(t, data.expected, shouldReconcile)
			})

			t.Run("delete event", func(t *testing.T) {
				// given
				ev := event.DeleteEvent{
					Object: tc,
				}

				// when
				shouldReconcile := predicate.Delete(ev)

				// then
				assert.Equal(t, data.expected, shouldReconcile)
			})
		})
	}
}
