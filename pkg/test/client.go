package test

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/kubefed/pkg/apis"
)

// NewFakeClient creates a fake K8s client with ability to override specific Get/List/Create/Update/StatusUpdate/Delete functions
func NewFakeClient(t T, initObjs ...runtime.Object) *FakeClient {
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)
	client := fake.NewFakeClientWithScheme(s, initObjs...)
	return &FakeClient{Client: client, T: t}
}

type FakeClient struct {
	client.Client
	T                T
	MockGet          func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error
	MockList         func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error
	MockCreate       func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error
	MockUpdate       func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error
	MockPatch        func(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error
	MockStatusUpdate func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error
	MockStatusPatch  func(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error
	MockDelete       func(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error
	MockDeleteAllOf  func(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error
}

type mockStatusUpdate struct {
	mockUpdate func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error
	mockPatch  func(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error
}

func (m *mockStatusUpdate) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	return m.mockUpdate(ctx, obj, opts...)
}

func (m *mockStatusUpdate) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return m.mockPatch(ctx, obj, patch, opts...)
}

func (c *FakeClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	if c.MockGet != nil {
		return c.MockGet(ctx, key, obj)
	}
	return c.Client.Get(ctx, key, obj)
}

func (c *FakeClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	if c.MockList != nil {
		return c.MockList(ctx, list, opts...)
	}
	return c.Client.List(ctx, list, opts...)
}

func (c *FakeClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	if c.MockCreate != nil {
		return c.MockCreate(ctx, obj, opts...)
	}

	// Set Generation to `1` for newly created objects since the kube fake client doesn't set it
	mt, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	mt.SetGeneration(1)
	return c.Client.Create(ctx, obj, opts...)
}

func (c *FakeClient) Status() client.StatusWriter {
	m := mockStatusUpdate{}
	if c.MockStatusUpdate == nil && c.MockStatusPatch == nil {
		return c.Client.Status()
	}
	if c.MockStatusUpdate != nil {
		m.mockUpdate = c.MockStatusUpdate
	}
	if c.MockStatusPatch != nil {
		m.mockPatch = c.MockStatusPatch
	}
	return &m
}

func (c *FakeClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	if c.MockUpdate != nil {
		return c.MockUpdate(ctx, obj, opts...)
	}

	// Update Generation if needed since the kube fake client doesn't update generations.
	// Increment the generation if spec (for objects with Spec) or data/stringData (for objects like CM and Secrets) is changed.
	updatingMeta, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	updatingMap, err := toMap(obj)
	if err != nil {
		return err
	}
	updatingMap["metadata"] = nil
	updatingMap["status"] = nil
	updatingMap["kind"] = nil
	updatingMap["apiVersion"] = nil

	current, err := cleanObject(obj)
	if err != nil {
		return err
	}
	if err := c.Client.Get(ctx, types.NamespacedName{Namespace: updatingMeta.GetNamespace(), Name: updatingMeta.GetName()}, current); err != nil {
		return err
	}
	currentMeta, err := meta.Accessor(current)
	if err != nil {
		return err
	}
	currentMap, err := toMap(current)
	if err != nil {
		return err
	}
	currentMap["metadata"] = nil
	currentMap["status"] = nil
	currentMap["kind"] = nil
	currentMap["apiVersion"] = nil

	if !reflect.DeepEqual(updatingMap, currentMap) {
		updatingMeta.SetGeneration(currentMeta.GetGeneration() + 1)
	} else {
		updatingMeta.SetGeneration(currentMeta.GetGeneration())
	}
	return c.Client.Update(ctx, obj, opts...)
}

func cleanObject(obj runtime.Object) (runtime.Object, error) {
	newObj := obj.DeepCopyObject()

	m, err := toMap(newObj)
	if err != nil {
		return nil, err
	}

	for k := range m {
		if k != "metadata" && k != "kind" && k != "apiVersion" {
			m[k] = nil
		}
	}

	return newObj, nil
}

func toMap(obj runtime.Object) (map[string]interface{}, error) {
	content, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(content, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func (c *FakeClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	if c.MockDelete != nil {
		return c.MockDelete(ctx, obj, opts...)
	}
	return c.Client.Delete(ctx, obj, opts...)
}

func (c *FakeClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	if c.MockDeleteAllOf != nil {
		return c.MockDeleteAllOf(ctx, obj, opts...)
	}
	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

func (c *FakeClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	if c.MockPatch != nil {
		return c.MockPatch(ctx, obj, patch, opts...)
	}
	return c.Client.Patch(ctx, obj, patch, opts...)
}
