package template

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// RetainNamespaces a func to retain only namespaces
	RetainNamespaces FilterFunc = func(obj runtime.RawExtension) bool {
		gvk := obj.Object.GetObjectKind().GroupVersionKind()
		return gvk.Kind == "Namespace"
	}

	// RetainAllButNamespaces a func to retain all but namespaces
	RetainAllButNamespaces FilterFunc = func(obj runtime.RawExtension) bool {
		gvk := obj.Object.GetObjectKind().GroupVersionKind()
		return gvk.Kind != "Namespace"
	}
)

// FilterFunc a function to retain an object or not
type FilterFunc func(runtime.RawExtension) bool

// Filter filters the given objs to return only those matching the given filters (if any)
// Accepts []runtime.RawExtension, []runtime.Object, or []runtimeclient.Object
func Filter(objs interface{}, filters ...FilterFunc) []runtime.RawExtension {
	// Convert input to []runtime.RawExtension
	var rawExtensions []runtime.RawExtension

	switch v := objs.(type) {
	case []runtime.RawExtension:
		rawExtensions = v
	case []runtime.Object, []runtimeclient.Object:
		// Handle both runtime.Object and runtimeclient.Object the same way
		// since runtimeclient.Object embeds runtime.Object
		var objects []runtime.Object
		if runtimeObjs, ok := v.([]runtime.Object); ok {
			objects = runtimeObjs
		} else {
			// Convert []runtimeclient.Object to []runtime.Object
			clientObjs := v.([]runtimeclient.Object)
			objects = make([]runtime.Object, len(clientObjs))
			for i, obj := range clientObjs {
				objects[i] = obj
			}
		}

		rawExtensions = make([]runtime.RawExtension, len(objects))
		for i, obj := range objects {
			rawExtensions[i] = runtime.RawExtension{Object: obj}
		}
	default:
		panic(fmt.Sprintf("unsupported type %T for Filter function. Supported types: []runtime.RawExtension, []runtime.Object, []runtimeclient.Object", objs))
	}

	// Apply filters - single implementation for all types
	result := make([]runtime.RawExtension, 0, len(rawExtensions))
loop:
	for _, obj := range rawExtensions {
		for _, filter := range filters {
			if !filter(obj) {
				continue loop
			}
		}
		result = append(result, obj)
	}

	return result
}
