package client

import (
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CompareToolchainObjects is a function that takes two instances of ToolchainObjects and compares them if their desired state is same
type CompareToolchainObjects func(firstObject, secondObject ToolchainObject) (bool, error)

// ToolchainObject is a type containing runtime.Object and information about it. It provides helpful methods on top of the object's data
type ToolchainObject interface {
	v1.Object
	GetGvk() schema.GroupVersionKind
	GetRuntimeObject() runtime.Object
	HasSameGvk(otherObject ToolchainObject) bool
	HasSameName(otherObject ToolchainObject) bool
	HasSameGvkAndName(otherObject ToolchainObject) bool
}

// ComparableToolchainObject is a ToolchainObject providing a method to compare it with another instance of ToolchainObject
type ComparableToolchainObject interface {
	ToolchainObject
	IsSame(otherObject ToolchainObject) (bool, error)
}

type toolchainObjectImpl struct {
	v1.Object
	gvk           schema.GroupVersionKind
	runtimeObject runtime.Object
}

// NewToolchainObject returns an instance of ToolchainObject for the given runtime.Object
func NewToolchainObject(ob runtime.Object) (ToolchainObject, error) {
	if ob == nil {
		return nil, fmt.Errorf("the provided object is nil, so the constructor cannot create an instance of ToolchainObject")
	}
	accessor, err := meta.Accessor(ob)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get accessor of object %v", ob)
	}

	return &toolchainObjectImpl{
		Object:        accessor,
		gvk:           ob.GetObjectKind().GroupVersionKind(),
		runtimeObject: ob,
	}, nil
}

// GetGvk returns GVK of the runtime.Object stored in ToolchainObject
func (o *toolchainObjectImpl) GetGvk() schema.GroupVersionKind {
	return o.gvk
}

// GetRuntimeObject returns the runtime.Object stored in ToolchainObject
func (o *toolchainObjectImpl) GetRuntimeObject() runtime.Object {
	return o.runtimeObject
}

// HasSameGvk returns if the provided ToolchainObject has the same GVK
func (o *toolchainObjectImpl) HasSameGvk(otherObject ToolchainObject) bool {
	return o.gvk == otherObject.GetGvk()
}

// HasSameName returns if the provided ToolchainObject has the same name
func (o *toolchainObjectImpl) HasSameName(otherObject ToolchainObject) bool {
	return o.GetName() == otherObject.GetName()
}

// HasSameGvkAndName returns if the provided ToolchainObject has the same GVK and name
func (o *toolchainObjectImpl) HasSameGvkAndName(otherObject ToolchainObject) bool {
	return o.HasSameGvk(otherObject) && o.HasSameName(otherObject)
}

type comparableToolchainObjectImpl struct {
	ToolchainObject
	compare CompareToolchainObjects
}

// NewComparableToolchainObject returns an instance of ComparableToolchainObject for the given runtime.Object
func NewComparableToolchainObject(ob runtime.Object, compare CompareToolchainObjects) (ComparableToolchainObject, error) {
	toolchainObject, err := NewToolchainObject(ob)
	if err != nil {
		return nil, err
	}
	return &comparableToolchainObjectImpl{
		ToolchainObject: toolchainObject,
		compare:         compare,
	}, nil
}

func (o *comparableToolchainObjectImpl) IsSame(otherObject ToolchainObject) (bool, error) {
	return o.compare(o.ToolchainObject, otherObject)
}
