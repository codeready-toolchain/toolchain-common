package client

import (
	"context"

	"github.com/codeready-toolchain/toolchain-common/pkg/audit"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	runtimeclient.Reader
	Scheme() *runtime.Scheme
	Create(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.CreateOption) error
	Delete(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.DeleteOption) error
	Patch(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, patch runtimeclient.Patch, opts ...runtimeclient.PatchOption) error
	Update(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.UpdateOption) error

	Status() AuditStatusClient
}

type client struct {
	cl runtimeclient.Client
}

func NewClient(cl runtimeclient.Client) Client {
	return client{
		cl: cl,
	}
}

func NewClientFromConfig(config *rest.Config, options runtimeclient.Options) (Client, error) {
	cl, err := runtimeclient.New(config, options)
	return client{
		cl: cl,
	}, err
}

var _ Client = client{}

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
// obj must be a struct pointer so that obj can be updated with the response
// returned by the Server.
func (c client) Get(ctx context.Context, key runtimeclient.ObjectKey, obj runtimeclient.Object, opts ...runtimeclient.GetOption) error {
	return c.cl.Get(ctx, key, obj, opts...)
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (c client) List(ctx context.Context, list runtimeclient.ObjectList, opts ...runtimeclient.ListOption) error {
	return c.cl.List(ctx, list, opts...)
}

// Scheme returns the scheme this client is using.
func (c client) Scheme() *runtime.Scheme {
	return c.cl.Scheme()
}

// Create saves the object obj in the Kubernetes cluster.
func (c client) Create(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.CreateOption) error {
	if err := c.cl.Create(ctx, obj, opts...); err != nil {
		return err
	}
	audit.LogAPIResourceChangeEvent(logger, obj, audit.ResourceCreated)
	return nil
}

// Delete deletes the given obj from Kubernetes cluster.
func (c client) Delete(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.DeleteOption) error {
	if err := c.cl.Delete(ctx, obj, opts...); err != nil {
		return err
	}
	audit.LogAPIResourceChangeEvent(logger, obj, audit.ResourceDeleted)
	return nil
}

// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c client) Update(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.UpdateOption) error {
	if err := c.cl.Update(ctx, obj, opts...); err != nil {
		return err
	}
	audit.LogAPIResourceChangeEvent(logger, obj, audit.ResourceCreated)
	return nil
}

// Patch patches the given obj in the Kubernetes cluster. obj must be a struct pointer so that obj can be updated with the content returned by the Server.
func (c client) Patch(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, patch runtimeclient.Patch, opts ...runtimeclient.PatchOption) error {
	if err := c.cl.Patch(ctx, obj, patch, opts...); err != nil {
		return err
	}
	audit.LogAPIResourceChangeEvent(logger, obj, audit.ResourcePatched)
	return nil
}

func (c client) Status() AuditStatusClient {
	return auditStatusClient{
		cl: c.cl.Status(),
	}
}

type AuditStatusClient interface {
	Update(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.UpdateOption) error
	Patch(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, patch runtimeclient.Patch, opts ...runtimeclient.PatchOption) error
}

type auditStatusClient struct {
	cl runtimeclient.StatusWriter
}

// Update updates the fields corresponding to the status subresource for the
// given obj. obj must be a struct pointer so that obj can be updated
// with the content returned by the Server.
func (c auditStatusClient) Update(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.UpdateOption) error {
	if err := c.cl.Update(ctx, obj, opts...); err != nil {
		return err
	}
	audit.LogAPIResourceChangeEvent(logger, obj, audit.ResourceStatusUpdated)
	return nil
}

// Patch patches the given object's subresource. obj must be a struct pointer so that obj can be updated with the content returned by the Server.
func (c auditStatusClient) Patch(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, patch runtimeclient.Patch, opts ...runtimeclient.PatchOption) error {
	if err := c.cl.Patch(ctx, obj, patch, opts...); err != nil {
		return err
	}
	audit.LogAPIResourceChangeEvent(logger, obj, audit.ResourceStatusPatched)
	return nil
}
