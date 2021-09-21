package client

import (
	"fmt"
	"sort"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type CompareToolchainObjects func(firstObject, secondObject runtimeclient.Object) (bool, error)

// SortToolchainObjectsByName takes the given list of ComparableToolchainObjects and sorts them by
// their namespaced name (it joins the object's namespace and name by a coma and compares them)
// The resulting sorted array is then returned.
// This function is important for write predictable and reliable tests
func SortToolchainObjectsByName(objects []runtimeclient.Object) []runtimeclient.Object {
	names := make([]string, len(objects))
	for i, object := range objects {
		names[i] = fmt.Sprintf("%s,%s", object.GetNamespace(), object.GetName())
	}
	sort.Strings(names)
	toolchainObjects := make([]runtimeclient.Object, len(objects))
	for i, name := range names {
		for _, object := range objects {
			if fmt.Sprintf("%s,%s", object.GetNamespace(), object.GetName()) == name {
				toolchainObjects[i] = object
				break
			}
		}
	}
	return toolchainObjects
}
