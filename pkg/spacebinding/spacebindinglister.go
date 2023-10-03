package spacebinding

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	errs "github.com/pkg/errors"
)

// Lister allows to list all spacebindings for a given space.
type Lister struct {
	ListSpaceBindingsFunc func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error)
	GetSpaceFunc          func(spaceName string) (*toolchainv1alpha1.Space, error)
}

func NewLister(listSpaceBindingsFunc func(spaceName string) ([]toolchainv1alpha1.SpaceBinding, error), getSpaceFunc func(spaceName string) (*toolchainv1alpha1.Space, error)) *Lister {
	return &Lister{
		ListSpaceBindingsFunc: listSpaceBindingsFunc,
		GetSpaceFunc:          getSpaceFunc,
	}
}

// ListForSpace it recursively searches up. It returns all the SBs from this space and from all the parent spaces of that space.
// It doesn't search SBs in the child spaces.
func (l *Lister) ListForSpace(space *toolchainv1alpha1.Space, foundBindings []toolchainv1alpha1.SpaceBinding) ([]toolchainv1alpha1.SpaceBinding, error) {
	parentBindings, err := l.ListSpaceBindingsFunc(space.Name)
	if err != nil {
		return foundBindings, err
	}

	// spaceBindings is the list that will be returned, it will contain either parent and child merged or just the "parent" ones retrieved above.
	foundBindings = mergeSpaceBindings(foundBindings, parentBindings)

	// no parent space,
	// let's return list of bindings accumulated since here ...
	if space.Spec.ParentSpace == "" {
		return foundBindings, nil
	}

	// fetch parent space and recursively keep going ...
	parentSpace, err := l.GetSpaceFunc(space.Spec.ParentSpace)
	if err != nil {
		// Error reading the object
		return foundBindings, errs.Wrap(err, "unable to get parent-space")
	}

	return l.ListForSpace(parentSpace, foundBindings)
}

func mergeSpaceBindings(foundBindings []toolchainv1alpha1.SpaceBinding, parentBindings []toolchainv1alpha1.SpaceBinding) []toolchainv1alpha1.SpaceBinding {
	// if both list are not empty, we have to merge them.
	// roles for the same username on SpaceBindings will override those on parentSpaceBindings,
	// so let's remove them from parentSpaceBinding before merging the two lists.
	for _, spaceBinding := range foundBindings {
		// iterate from back to front, so you don't have to worry about indexes that are deleted.
		for i := len(parentBindings) - 1; i >= 0; i-- {
			if parentBindings[i].Spec.MasterUserRecord == spaceBinding.Spec.MasterUserRecord {
				parentBindings = append(parentBindings[:i], parentBindings[i+1:]...)
				break
			}
		}
	}
	// merge lists now that there are no duplicates
	// and just use the TypeMeta and ListMeta objects from the spaceBinding list
	// since only the Items field is relevant from this object
	return append(foundBindings, parentBindings...)
}
