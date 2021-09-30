package client

import (
	"fmt"
	"sort"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SortObjectsByName takes the given list of Objects and sorts them by
// their namespaced name (it joins the object's namespace and name by a coma and compares them)
// The resulting sorted array is then returned.
// This function is important for write predictable and reliable tests
func SortObjectsByName(objects []runtimeclient.Object) []runtimeclient.Object {
	names := make([]string, len(objects))
	for i, object := range objects {
		names[i] = fmt.Sprintf("%s,%s", object.GetNamespace(), object.GetName())
	}
	sort.Strings(names)
	sortedObjects := make([]runtimeclient.Object, len(objects))
	for i, name := range names {
		for _, object := range objects {
			if fmt.Sprintf("%s,%s", object.GetNamespace(), object.GetName()) == name {
				sortedObjects[i] = object
				break
			}
		}
	}
	return sortedObjects
}

// SameGVKandName returns `true` if both objects have the same GroupVersionKind and Name, `false` otherwise
func SameGVKandName(a, b runtimeclient.Object) bool {
	return a.GetObjectKind().GroupVersionKind() == b.GetObjectKind().GroupVersionKind() &&
		a.GetName() == b.GetName()
}
