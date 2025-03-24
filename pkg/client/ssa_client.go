package client

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SsaApplyClient the client to use when creating or updating objects. It uses SSA to apply the objects
// to the cluster and takes care of migrating the objects from ordinary "CRUD" flow to SSA.
type SsaApplyClient struct {
	Client client.Client

	// NonSSAFieldOwner is a the field owner that is used by the operations that do not set the field owner explicitly.
	//
	// If you don't use an explicit field owner in your CRUD operations, set this to the value obtained from GetDefaultFieldOwner.
	NonSSAFieldOwner string

	// The field owner to use for SSA-applied objects.
	FieldOwner string
}

// NewSsaApplyClient constructs a new apply client using the client and rest config of the manager.
func NewSsaApplyClient(mgr ctrl.Manager, fieldOwner string) *SsaApplyClient {
	return &SsaApplyClient{
		Client:           mgr.GetClient(),
		NonSSAFieldOwner: GetDefaultFieldOwner(mgr.GetConfig()),
		FieldOwner:       fieldOwner,
	}
}

// GetDefaultFieldOwner returns the default field owner that is applied if no explicit field owner is set.
// This can be used to determine the field owner used by the non-SSA CRUD operations performed by
// the kubernetes client.
//
// This value is obtained from the user agent header defined in the provided config, or, if it is not set,
// from the default kubernetes user agent string.
//
// If the provided config is nil, the default kubernetes user agent is returned
func GetDefaultFieldOwner(cfg *rest.Config) string {
	var ua string
	if cfg != nil {
		ua = cfg.UserAgent
	}

	if len(ua) == 0 {
		ua = rest.DefaultKubernetesUserAgent()
	}

	// the default field owner is the name of the binary, which is also
	// the first portion of the default User-Agent.
	name := strings.Split(ua, "/")[0]
	return name
}

type ssaApplyObjectConfiguration struct {
	owner     metav1.Object
	newLabels map[string]string
	skipIf    func(client.Object) bool
}

func newSsaApplyObjectConfiguration(options ...SsaApplyObjectOption) ssaApplyObjectConfiguration {
	config := ssaApplyObjectConfiguration{}
	for _, apply := range options {
		apply(&config)
	}
	return config
}

// SsaApplyObjectOption an option when creating or updating a resource
type SsaApplyObjectOption func(*ssaApplyObjectConfiguration)

// SetOwnerReference sets the owner reference of the resource (default: `nil`)
func SetOwnerReference(owner metav1.Object) SsaApplyObjectOption {
	return func(config *ssaApplyObjectConfiguration) {
		config.owner = owner
	}
}

// SkipIf will cause the apply function skip the update of the object if
// the provided function returns true. The supplied object is guaranteed to
// have its GVK set.
func SkipIf(test func(client.Object) bool) SsaApplyObjectOption {
	return func(config *ssaApplyObjectConfiguration) {
		config.skipIf = test
	}
}

// EnsureLabels makes sure that the provided labels are applied to the object even if
// the supplied object doesn't have them set.
func EnsureLabels(labels map[string]string) SsaApplyObjectOption {
	return func(config *ssaApplyObjectConfiguration) {
		config.newLabels = labels
	}
}

// Configure sets the owner reference and merges the labels. Other options modify the logic
// of apply function and therefore need to be checked manually.
func (c *ssaApplyObjectConfiguration) Configure(obj client.Object, s *runtime.Scheme) error {
	if c.owner != nil {
		if err := controllerutil.SetControllerReference(c.owner, obj, s); err != nil {
			return err
		}
	}
	MergeLabels(obj, c.newLabels)

	return nil
}

// ApplyObject creates the object if is missing or update it if it already exists using an SSA patch.
func (c *SsaApplyClient) ApplyObject(ctx context.Context, obj client.Object, options ...SsaApplyObjectOption) error {
	config := newSsaApplyObjectConfiguration(options...)
	if err := config.Configure(obj, c.Client.Scheme()); err != nil {
		return composeError(obj, fmt.Errorf("failed to configure the apply function: %w", err))
	}

	if err := prepareForSSA(obj, c.Client.Scheme()); err != nil {
		return composeError(obj, fmt.Errorf("failed to prepare the object for SSA: %w", err))
	}

	if config.skipIf != nil && config.skipIf(obj) {
		return nil
	}

	if err := c.Client.Patch(ctx, obj, client.Apply, client.FieldOwner(c.FieldOwner), client.ForceOwnership); err != nil {
		return composeError(obj, err)
	}

	return nil
}

func composeError(obj client.Object, err error) error {
	return fmt.Errorf("unable to patch '%s' called '%s' in namespace '%s': %w", obj.GetObjectKind().GroupVersionKind(), obj.GetName(), obj.GetNamespace(), err)
}

func prepareForSSA(obj client.Object, scheme *runtime.Scheme) error {
	obj.SetManagedFields(nil)
	return EnsureGVK(obj, scheme)
}

// EnsureGVK makes sure that the object has the GVK set.
//
// If the GVK is empty, it will consult the scheme.
func EnsureGVK(obj client.Object, scheme *runtime.Scheme) error {
	var empty schema.GroupVersionKind

	if obj.GetObjectKind().GroupVersionKind() != empty {
		return nil
	}

	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return err
	}
	obj.GetObjectKind().SetGroupVersionKind(gvk)

	return nil
}

// Apply is a utility function that just calls `ApplyObject` in a loop on all the supplied objects.
func (c *SsaApplyClient) Apply(ctx context.Context, toolchainObjects []client.Object, opts ...SsaApplyObjectOption) error {
	for _, toolchainObject := range toolchainObjects {
		if err := c.ApplyObject(ctx, toolchainObject, opts...); err != nil {
			return err
		}
	}
	return nil
}
