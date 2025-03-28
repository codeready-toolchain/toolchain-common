package client

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SSAApplyClient the client to use when creating or updating objects. It uses SSA to apply the objects
// to the cluster.
//
// It doesn't try to migrate the objects from ordinary "CRUD" flow to SSA to be as efficient as possible.
// If you need to do that check k8s.io/client-go/util/csaupgrade.UpgradeManagedFields().
type SSAApplyClient struct {
	Client client.Client

	// The field owner to use for SSA-applied objects.
	FieldOwner string
}

// NewSSAApplyClient creates a new SSAApplyClient from the provided parameters that will use the provided field owner
// for the patches.
func NewSSAApplyClient(cl client.Client, fieldOwner string) *SSAApplyClient {
	return &SSAApplyClient{
		Client:     cl,
		FieldOwner: fieldOwner,
	}
}

type ssaApplyObjectConfiguration struct {
	owner     metav1.Object
	newLabels map[string]string
	skipIf    func(client.Object) bool
}

func newSSAApplyObjectConfiguration(options ...SSAApplyObjectOption) ssaApplyObjectConfiguration {
	config := ssaApplyObjectConfiguration{}
	for _, apply := range options {
		apply(&config)
	}
	return config
}

// SSAApplyObjectOption an option when creating or updating a resource
type SSAApplyObjectOption func(*ssaApplyObjectConfiguration)

// SetOwnerReference sets the owner reference of the resource (default: `nil`)
func SetOwnerReference(owner metav1.Object) SSAApplyObjectOption {
	return func(config *ssaApplyObjectConfiguration) {
		config.owner = owner
	}
}

// SkipIf will cause the apply function skip the update of the object if
// the provided function returns true. The supplied object is guaranteed to
// have its GVK set.
func SkipIf(test func(client.Object) bool) SSAApplyObjectOption {
	return func(config *ssaApplyObjectConfiguration) {
		config.skipIf = test
	}
}

// EnsureLabels makes sure that the provided labels are applied to the object even if
// the supplied object doesn't have them set.
func EnsureLabels(labels map[string]string) SSAApplyObjectOption {
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
func (c *SSAApplyClient) ApplyObject(ctx context.Context, obj client.Object, options ...SSAApplyObjectOption) error {
	config := newSSAApplyObjectConfiguration(options...)
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
	message := "unable to patch '%s' called '%s' in namespace '%s': %w"
	if !obj.GetObjectKind().GroupVersionKind().Empty() {
		return fmt.Errorf(message, obj.GetObjectKind().GroupVersionKind(), obj.GetName(), obj.GetNamespace(), err)
	} else {
		return fmt.Errorf(message, reflect.TypeOf(obj), obj.GetName(), obj.GetNamespace(), err)
	}
}

func prepareForSSA(obj client.Object, scheme *runtime.Scheme) error {
	// Managed fields need to be set to nil when doing the SSA apply.
	// This will not overwrite the field in the cluster - managed fields
	// is treated specially by the api server so that clients that do not
	// set it, don't cause its deletion.
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
func (c *SSAApplyClient) Apply(ctx context.Context, toolchainObjects []client.Object, opts ...SSAApplyObjectOption) error {
	for _, toolchainObject := range toolchainObjects {
		if err := c.ApplyObject(ctx, toolchainObject, opts...); err != nil {
			return err
		}
	}
	return nil
}
