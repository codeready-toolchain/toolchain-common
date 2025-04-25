package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/csaupgrade"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/value"

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

	// NonSSAFieldOwner should be set to the same value as the user agent used by the provided Kubernetes client
	// or to the value of the explicit field owner that the calling code used to use with the normal CRUD operations
	// (highly unlikely and not the case in our codebase).
	//
	// The user agent can be obtained from the REST config from which the client is constructed.
	//
	// The user agent in the REST config is usually empty, so there's no need to set it here either in that case.
	NonSSAFieldOwner string

	// DefaultBehavior specifies the default behavior of the SSA apply client
	DefaultBehavior SSAApplyClientDefaultBehavior
}

type SSAApplyClientDefaultBehavior struct {
	// MigrateSSA specifies the default SSA migration behavior.
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
	MigrateSSA bool

	// EnsureExclusiveFieldOwnership specifies whether the client makes sure the field owner owns the fields in the
	// object as it exists exclusively prior to applying the SSA patch.
	//
	// This is only important for objects that can be modified by "untrusted" parties - i.e. objects that the users
	// are given the ability to modify yet we still need to make sure that all the fields that are originating from
	// our templates have values that we declare in the templates. Because the user has the write access also
	// to the managed fields of those objects they could be able to arrange the managed fields such that the SSA
	// apply would leave the the user-defined values "in" even if they were removed from our templates.
	//
	//
	// Note that this is only necessary for objects that are meant to be user-editable. We generally don't need to use
	// this on objects that should only ever be updated by us or other Kubernetes controllers, because those are
	// inherently trusted.
	EnsureExclusiveFieldOwnership bool
}

// NewSSAApplyClient creates a new SSAApplyClient from the provided parameters that will use the provided field owner
// for the patches.
//
// The returned client checks for the SSA migration and does NOT check for exclusive field ownership by default.
func NewSSAApplyClient(cl client.Client, fieldOwner string) *SSAApplyClient {
	return &SSAApplyClient{
		Client:     cl,
		FieldOwner: fieldOwner,
		DefaultBehavior: SSAApplyClientDefaultBehavior{
			MigrateSSA: true,
		},
	}
}

type ssaApplyObjectConfiguration struct {
	ownerReference metav1.Object
	newLabels      map[string]string
	skipIf         func(client.Object) bool

	// pointer-to-bool to model 3 states - not specified, true or false
	migrateSSA                    *bool
	ensureExclusiveFieldOwnership *bool
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
		config.ownerReference = owner
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
		config.migrateSSA = ptr.To(value)
	}
}

// EnsureExclusiveFieldOwnership instructs the Apply to ensure that all fields that are being
// applied are exclusively owned by the expected field owner in the managed fields. This can be
// needed in cases where other users are expected to have write access to the objects we manage
// and we assume that it is 100% necessary for the objects to have exactly the "shape" we require
// on the fields that we set.
func EnsureExclusiveFieldOwnership(value bool) SSAApplyObjectOption {
	return func(config *ssaApplyObjectConfiguration) {
		config.ensureExclusiveFieldOwnership = ptr.To(value)
	}
}

// Configure sets the owner reference and merges the labels. Other options modify the logic
// of apply function and therefore need to be checked manually.
func (c *ssaApplyObjectConfiguration) Configure(obj client.Object, s *runtime.Scheme) error {
	if c.ownerReference != nil {
		if err := controllerutil.SetControllerReference(c.ownerReference, obj, s); err != nil {
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

	migrateSSA := isTrueOrDefault(config.migrateSSA, c.DefaultBehavior.MigrateSSA)
	ensureExclusiveFieldOwnership := isTrueOrDefault(config.ensureExclusiveFieldOwnership, c.DefaultBehavior.EnsureExclusiveFieldOwnership)

	if err := c.prepareInCluster(ctx, obj, migrateSSA, ensureExclusiveFieldOwnership); err != nil {
		return composeError(obj,
			fmt.Errorf("failed to prepare the object in cluster. migration required: %v, exclusive ownership required: %v",
				migrateSSA, config.ensureExclusiveFieldOwnership))
	}

	if config.skipIf != nil && config.skipIf(obj) {
		return nil
	}

	if err := c.Client.Patch(ctx, obj, client.Apply, client.FieldOwner(c.FieldOwner), client.ForceOwnership); err != nil {
		return composeError(obj, err)
	}

	return nil
}

func isTrueOrDefault(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func (c *SSAApplyClient) prepareInCluster(ctx context.Context, obj client.Object, migrateSSA bool, ensureExclusiveOwnership bool) error {
	inCluster := obj.DeepCopyObject().(client.Object)
	if err := c.Client.Get(ctx, client.ObjectKeyFromObject(obj), inCluster); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get the object from the cluster while preparing the object for SSA apply: %w", err)
		}

		// nothing to do if the object is not yet present in the cluster
		return nil
	}

	updatedBecauseOfMigration := false
	updatedBecauseOfExclusiveOwnership := false
	var err error

	if migrateSSA {
		if updatedBecauseOfMigration, err = c.migrateSSA(inCluster); err != nil {
			return err
		}
	}

	if ensureExclusiveOwnership {
		objectFields, err := getObjectFields(obj)
		if err != nil {
			return err
		}
		updatedBecauseOfExclusiveOwnership, err = ensureExclusiveFieldOwnership(inCluster, c.FieldOwner, objectFields)
		if err != nil {
			return err
		}
	}

	if updatedBecauseOfMigration || updatedBecauseOfExclusiveOwnership {
		return c.Client.Update(ctx, inCluster)
	}

	return nil
}

func (c *SSAApplyClient) migrateSSA(obj client.Object) (bool, error) {
	oldFieldOwner := c.NonSSAFieldOwner
	if len(oldFieldOwner) == 0 {
		// this is how the kubernetes api server determines the default owner from the user agent
		// The default user agent has the form of "name-of-binary/version information etc.".
		// The owner is the first part of the UA unless explicitly specified in the request URI.
		oldFieldOwner = strings.Split(rest.DefaultKubernetesUserAgent(), "/")[0]
	}
	if isSsaMigrationNeeded(obj, oldFieldOwner, c.FieldOwner) {
		if err := migrateToSSA(obj, oldFieldOwner, c.FieldOwner); err != nil {
			return false, fmt.Errorf("failed to migrate the managed fields: %w", err)
		}
		return true, nil
	}

	return false, nil
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

func isSsaMigrationNeeded(obj client.Object, expectedOwners ...string) bool {
	for _, mf := range obj.GetManagedFields() {
		if slices.Contains(expectedOwners, mf.Manager) && mf.Operation != metav1.ManagedFieldsOperationApply {
			return true
		}
	}
	return false
}

func migrateToSSA(obj client.Object, oldFieldOwner, newFieldOwner string) error {
	return csaupgrade.UpgradeManagedFields(obj, sets.New(oldFieldOwner, newFieldOwner), newFieldOwner)
}

// ensureExclusiveFieldOwnership makes sure that if the provided owner owns all the required fields in
// the provided object, it is the sole owner of those fields (by updating the managed fields
// of the object).
//
// Once the owner is the sole owner of all the required fields, it can be sure that the SSA patch
// handles the deletes in the expected manner.
func ensureExclusiveFieldOwnership(obj client.Object, fieldOwner string, requiredFields *fieldpath.Set) (bool, error) {
	mfs := obj.GetManagedFields()
	if len(mfs) == 0 {
		return false, nil
	}

	indicesToRemove := []int{}
	addEntryForRequired := true
	apiVersion := ""
	modified := false
	for i := range mfs {
		// a quick and hacky way of getting the API version by copying what's in the other fields. The API version is not set
		// on a client.Object and we'd need to get it somehow from the scheme or even from the rest mapper. So let's just assume
		// that the object exists only in a single version in the cluster (which is true in the vast majority of the cases).
		apiVersion = mfs[i].APIVersion
		fieldPaths, err := decodeManagedFieldsEntrySet(mfs[i])
		if err != nil {
			return false, err
		}
		remainderOfFields := fieldPaths.Difference(requiredFields)
		if remainderOfFields.Empty() {
			if fieldOwner == mfs[i].Manager && fieldPaths.Size() == requiredFields.Size() {
				addEntryForRequired = false
				if mfs[i].Operation != metav1.ManagedFieldsOperationApply {
					mfs[i].Operation = metav1.ManagedFieldsOperationApply
					modified = true
				}
			} else {
				indicesToRemove = append(indicesToRemove, i)
				modified = true // will be modified below
			}
		} else {
			if err := encodeManagedFieldsEntrySet(&mfs[i], remainderOfFields); err != nil {
				return false, err
			}
			modified = true
		}
	}

	newManagedFields := []metav1.ManagedFieldsEntry{}
	startIdx := 0
	for _, stopIdx := range indicesToRemove {
		newManagedFields = append(newManagedFields, mfs[startIdx:stopIdx]...)
		startIdx = stopIdx + 1
		if startIdx >= len(mfs) {
			break
		}
	}
	if startIdx < len(mfs) {
		newManagedFields = append(newManagedFields, mfs[startIdx:]...)
	}

	if addEntryForRequired {
		modified = true
		entry := metav1.ManagedFieldsEntry{
			Manager:    fieldOwner,
			Operation:  metav1.ManagedFieldsOperationApply,
			FieldsType: "FieldsV1",
			APIVersion: apiVersion,
			FieldsV1:   &metav1.FieldsV1{},
		}
		if err := encodeManagedFieldsEntrySet(&entry, requiredFields); err != nil {
			return modified, err
		}
		newManagedFields = append(newManagedFields, entry)
	}

	if modified {
		obj.SetManagedFields(newManagedFields)
	}

	return modified, nil
}

func decodeManagedFieldsEntrySet(f metav1.ManagedFieldsEntry) (s fieldpath.Set, err error) {
	err = s.FromJSON(bytes.NewReader(f.FieldsV1.Raw))
	return s, err
}

func encodeManagedFieldsEntrySet(f *metav1.ManagedFieldsEntry, s *fieldpath.Set) (err error) {
	f.FieldsV1.Raw, err = s.ToJSON()
	return err
}

func getObjectFields(obj client.Object) (fields *fieldpath.Set, err error) {
	var data []byte
	data, err = json.Marshal(obj)
	if err != nil {
		return
	}
	var val value.Value
	val, err = value.FromJSON(data)
	if err != nil {
		return
	}
	// clear up the fields that cannot be updated using the SSA patch.
	val.AsMap().Delete("kind")
	val.AsMap().Delete("apiVersion")
	val.AsMap().Delete("status")
	metadata, present := val.AsMap().Get("metadata")
	if present {
		// only labels and annotations are updatable in the metadata
		labels, labelsOk := metadata.AsMap().Get("labels")
		annotations, annosOk := metadata.AsMap().Get("annotations")
		newMetadata := value.NewValueInterface(map[string]any{})
		if labelsOk {
			newMetadata.AsMap().Set("labels", labels)
		}
		if annosOk {
			newMetadata.AsMap().Set("annotations", annotations)
		}
		val.AsMap().Set("metadata", newMetadata)
	}

	fields = fieldpath.SetFromValue(val)
	return
}
