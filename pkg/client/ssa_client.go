package client

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/csaupgrade"
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

	// MigrateSSAByDefault specifies the default SSA migration behavior.
	//
	// When checking for the migration, there is an additional GET of the resource, followed by optional
	// UPDATE (if the migration is needed) before the actual changes to the objects are applied.
	//
	// This field specifies the default behavior that can be overridden by supplying an explicit MigrateSSA() option
	// to ApplyObject or Apply methods.
	//
	// The main advantage of using the SSA in our code is that ability of SSA to handle automatic deletion of fields
	// that we no longer set in our templates. But this only works when the fields are owned by managers and applied
	// using "Apply" operation. As long as there is an "Update" entry with given field (even if the owner is the same)
	// the field WILL NOT be automatically deleted by Kubernetes.
	//
	// Therefore, we need to make sure that our manager uses ONLY the Apply operations. This maximizes the chance
	// that the object will look the way we need.
	MigrateSSAByDefault bool

	// NonSSAFieldOwner should be set to the same value as the user agent used by the provided Kubernetes client
	// or to the value of the explicit field owner that the calling code used to use with the normal CRUD operations
	// (highly unlikely and not the case in our codebase).
	//
	// The user agent can be obtained from the REST config from which the client is constructed.
	//
	// The user agent in the REST config is usually empty, so there's no need to set it here either in that case.
	NonSSAFieldOwner string
}

// NewSSAApplyClient creates a new SSAApplyClient from the provided parameters that will use the provided field owner
// for the patches.
//
// The returned client checks for the SSA migration by default.
func NewSSAApplyClient(cl client.Client, fieldOwner string) *SSAApplyClient {
	return &SSAApplyClient{
		Client:              cl,
		FieldOwner:          fieldOwner,
		MigrateSSAByDefault: true,
	}
}

type migrateSSA int

const (
	migrateSSANotSpecified migrateSSA = iota
	migrateSSAYes
	migrateSSANo
)

type ssaApplyObjectConfiguration struct {
	owner      metav1.Object
	newLabels  map[string]string
	skipIf     func(client.Object) bool
	migrateSSA migrateSSA
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

// MigrateSSA instructs the apply to do the SSA managed fields migration or not.
// If not used at all, the MigrateSSAByDefault field of the SSA client determines
// whether the fields will be migrated or not.
func MigrateSSA(value bool) SSAApplyObjectOption {
	return func(config *ssaApplyObjectConfiguration) {
		if value {
			config.migrateSSA = migrateSSAYes
		} else {
			config.migrateSSA = migrateSSANo
		}
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

	if config.migrateSSA == migrateSSAYes || (config.migrateSSA == migrateSSANotSpecified && c.MigrateSSAByDefault) {
		if err := c.migrateSSA(ctx, obj); err != nil {
			return composeError(obj, err)
		}
	}

	if config.skipIf != nil && config.skipIf(obj) {
		return nil
	}

	if err := c.Client.Patch(ctx, obj, client.Apply, client.FieldOwner(c.FieldOwner), client.ForceOwnership); err != nil {
		return composeError(obj, err)
	}

	return nil
}

func (c *SSAApplyClient) migrateSSA(ctx context.Context, obj client.Object) error {
	orig := obj.DeepCopyObject().(client.Object)
	if err := c.Client.Get(ctx, client.ObjectKeyFromObject(obj), orig); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get the object from the cluster while migrating managed fields: %w", err)
		}
		orig = nil
	}

	if orig != nil {
		oldFieldOwner := c.NonSSAFieldOwner
		if len(oldFieldOwner) == 0 {
			// this is how the kubernetes api server determines the default owner from the user agent
			// The default user agent has the form of "name-of-binary/version information etc.".
			// The owner is the first part of the UA unless explicitly specified in the request URI.
			oldFieldOwner = strings.Split(rest.DefaultKubernetesUserAgent(), "/")[0]
		}
		if isSsaMigrationNeeded(orig, oldFieldOwner) {
			if err := migrateToSSA(ctx, c.Client, orig, oldFieldOwner, c.FieldOwner); err != nil {
				return fmt.Errorf("failed to migrate the managed fields: %w", err)
			}
		}
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

func isSsaMigrationNeeded(obj client.Object, expectedOwner string) bool {
	for _, mf := range obj.GetManagedFields() {
		if mf.Manager == expectedOwner && mf.Operation != metav1.ManagedFieldsOperationApply {
			return true
		}
	}
	return false
}

func migrateToSSA(ctx context.Context, cl client.Client, obj client.Object, oldFieldOwner, newFieldOwner string) error {
	if err := csaupgrade.UpgradeManagedFields(obj, sets.New(oldFieldOwner), newFieldOwner); err != nil {
		return err
	}
	return cl.Update(ctx, obj)
}
