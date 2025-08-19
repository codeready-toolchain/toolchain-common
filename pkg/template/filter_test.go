package template_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Test fixtures - shared across all tests
var (
	ns1 = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "Namespace",
			"metadata": map[string]interface{}{
				"name": "ns1",
			},
		},
	}
	ns2 = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "Namespace",
			"metadata": map[string]interface{}{
				"name": "ns2",
			},
		},
	}
	rb1 = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "RoleBinding",
			"metadata": map[string]interface{}{
				"name": "rb1",
			},
		},
	}
	rb2 = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "RoleBinding",
			"metadata": map[string]interface{}{
				"name": "rb2",
			},
		},
	}
)

// filterTestCase represents a test case for filter testing
type filterTestCase struct {
	name            string
	objects         []runtime.Object
	filters         []template.FilterFunc
	expectedCount   int
	expectedObjects []runtime.Object
}

// runFilterTests runs a set of filter test cases for different input types
func runFilterTests(t *testing.T, testCases []filterTestCase) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with RawExtension slice
			t.Run("RawExtension slice", func(t *testing.T) {
				rawExtensions := make([]runtime.RawExtension, len(tc.objects))
				for i, obj := range tc.objects {
					rawExtensions[i] = runtime.RawExtension{Object: obj}
				}

				result := template.Filter(rawExtensions, tc.filters...)

				require.Len(t, result, tc.expectedCount)
				for i, expected := range tc.expectedObjects {
					assert.Equal(t, expected, result[i].Object)
				}
			})

			// Test with runtime.Object slice
			t.Run("runtime.Object slice", func(t *testing.T) {
				result := template.Filter(tc.objects, tc.filters...)

				require.Len(t, result, tc.expectedCount)
				for i, expected := range tc.expectedObjects {
					assert.Equal(t, expected, result[i].Object)
				}
			})

			// Test with runtimeclient.Object slice
			t.Run("runtimeclient.Object slice", func(t *testing.T) {
				clientObjects := make([]runtimeclient.Object, len(tc.objects))
				for i, obj := range tc.objects {
					clientObjects[i] = obj.(runtimeclient.Object)
				}

				result := template.Filter(clientObjects, tc.filters...)

				require.Len(t, result, tc.expectedCount)
				for i, expected := range tc.expectedObjects {
					assert.Equal(t, expected, result[i].Object)
				}
			})
		})
	}
}

func TestFilter(t *testing.T) {
	testCases := []filterTestCase{
		{
			name:            "no filter",
			objects:         []runtime.Object{ns1, rb1, ns2, rb2},
			filters:         []template.FilterFunc{},
			expectedCount:   4,
			expectedObjects: []runtime.Object{ns1, rb1, ns2, rb2},
		},
		{
			name:            "all filters (conflicting)",
			objects:         []runtime.Object{ns1, rb1, ns2, rb2},
			filters:         []template.FilterFunc{template.RetainNamespaces, template.RetainAllButNamespaces},
			expectedCount:   0,
			expectedObjects: []runtime.Object{},
		},
		{
			name:            "retain namespaces - single result",
			objects:         []runtime.Object{ns1, rb1},
			filters:         []template.FilterFunc{template.RetainNamespaces},
			expectedCount:   1,
			expectedObjects: []runtime.Object{ns1},
		},
		{
			name:            "retain namespaces - multiple results",
			objects:         []runtime.Object{ns1, rb1, ns2, rb2},
			filters:         []template.FilterFunc{template.RetainNamespaces},
			expectedCount:   2,
			expectedObjects: []runtime.Object{ns1, ns2},
		},
		{
			name:            "retain namespaces - no results",
			objects:         []runtime.Object{rb1, rb2},
			filters:         []template.FilterFunc{template.RetainNamespaces},
			expectedCount:   0,
			expectedObjects: []runtime.Object{},
		},
		{
			name:            "retain all but namespaces - single result",
			objects:         []runtime.Object{ns1, rb1},
			filters:         []template.FilterFunc{template.RetainAllButNamespaces},
			expectedCount:   1,
			expectedObjects: []runtime.Object{rb1},
		},
		{
			name:            "retain all but namespaces - multiple results",
			objects:         []runtime.Object{ns1, rb1, ns2, rb2},
			filters:         []template.FilterFunc{template.RetainAllButNamespaces},
			expectedCount:   2,
			expectedObjects: []runtime.Object{rb1, rb2},
		},
		{
			name:            "retain all but namespaces - no results",
			objects:         []runtime.Object{ns1, ns2},
			filters:         []template.FilterFunc{template.RetainAllButNamespaces},
			expectedCount:   0,
			expectedObjects: []runtime.Object{},
		},
	}

	runFilterTests(t, testCases)
}

func TestFilterUnsupportedType(t *testing.T) {
	t.Run("panic on unsupported type", func(t *testing.T) {
		// given
		unsupportedSlice := []string{"not", "supported"}

		// when/then
		assert.Panics(t, func() {
			template.Filter(unsupportedSlice)
		})
	})
}
