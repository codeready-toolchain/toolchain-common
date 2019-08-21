package test

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// NewFakeClient creates a fake K8s client with ability to override specific Get/List/Create/Update/StatusUpdate/Delete functions
func NewFakeClient(t *testing.T, initObjs ...runtime.Object) *FakeClient {
	client := fake.NewFakeClientWithScheme(scheme.Scheme, initObjs...)
	return &FakeClient{client, t, nil, nil, nil, nil, nil, nil}
}

type FakeClient struct {
	client.Client
	T                *testing.T
	MockGet          func(key client.ObjectKey, obj runtime.Object) error
	MockList         func(opts *client.ListOptions, list runtime.Object) error
	MockCreate       func(obj runtime.Object) error
	MockUpdate       func(obj runtime.Object) error
	MockStatusUpdate func(obj runtime.Object) error
	MockDelete       func(obj runtime.Object, opts ...client.DeleteOptionFunc) error
}

type mockStatusUpdate struct {
	mockUpdate func(obj runtime.Object) error
}

func (m *mockStatusUpdate) Update(ctx context.Context, obj runtime.Object) error {
	return m.mockUpdate(obj)
}

func (c *FakeClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	if c.MockGet != nil {
		return c.MockGet(key, obj)
	}
	return c.Client.Get(ctx, key, obj)
}

func (c *FakeClient) List(ctx context.Context, opts *client.ListOptions, list runtime.Object) error {
	if c.MockList != nil {
		return c.MockList(opts, list)
	}
	return c.Client.List(ctx, opts, list)
}

func (c *FakeClient) Create(ctx context.Context, obj runtime.Object) error {
	if c.MockCreate != nil {
		return c.MockCreate(obj)
	}
	return c.Client.Create(ctx, obj)
}

func (c *FakeClient) Status() client.StatusWriter {
	if c.MockStatusUpdate != nil {
		return &mockStatusUpdate{mockUpdate: c.MockStatusUpdate}
	}
	return c.Client.Status()
}

func (c *FakeClient) Update(ctx context.Context, obj runtime.Object) error {
	if c.MockUpdate != nil {
		return c.MockUpdate(obj)
	}
	return c.Client.Update(ctx, obj)
}

func (c *FakeClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOptionFunc) error {
	if c.MockDelete != nil {
		return c.MockDelete(obj, opts...)
	}
	return c.Client.Delete(ctx, obj, opts...)
}
