package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/csaupgrade"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplyClient the client to use when creating or updating objects. It uses SSA to apply the objects
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

	name := strings.Split(ua, "/")[0]
	return name
}

type ssaApplyObjectConfiguration struct {
	owner           metav1.Object
	newLabels       map[string]string
	determineUpdate bool
	skipIf          func(client.Object) bool
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

// SsaSetOwner sets the owner of the resource (default: `nil`)
func SsaSetOwner(owner metav1.Object) SsaApplyObjectOption {
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

// DetermineUpdate instructs the ApplyObject function to truly test whether the object
// was updated or not during the apply operation. By default, the function always returns
// true on success which is more efficient and sufficient in most circumstances.
func DetermineUpdate(value bool) SsaApplyObjectOption {
	return func(config *ssaApplyObjectConfiguration) {
		config.determineUpdate = value
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

// ApplyObject creates the object if is missing and if the owner object is provided, then it's set as a controller reference.
// If the objects exists then when the spec content has changed (based on the content of the annotation in the original object) then it
// is automatically updated. If it looks to be same then based on the value of forceUpdate param it updates the object or not.
// The return boolean says if the object was either created or updated (`true`). If nothing changed (ie, the generation was not
// incremented by the server), then it returns `false`.
//
// NOTE: the return boolean IS ALWAYS TRUE on success by default. This is much more efficient. If you truly need
// to determine whether the object has been updated or not (which is not the case anywhere in the codebase but the tests),
// you can use the DetermineUpdate() option.
func (c *SsaApplyClient) ApplyObject(ctx context.Context, obj client.Object, options ...SsaApplyObjectOption) (bool, error) {
	config := newSsaApplyObjectConfiguration(options...)
	if err := config.Configure(obj, c.Client.Scheme()); err != nil {
		return false, err
	}

	if err := prepareForSSA(obj, c.Client.Scheme()); err != nil {
		return false, err
	}

	if config.skipIf != nil && config.skipIf(obj) {
		return false, nil
	}

	updated := true

	// NOTE: once the SSA migration is not needed anymore, read the orig conditionally only when config.determineUpdate is true.

	// this cannot fail - we would have failed previously already because the scheme wouldn't have known the object's kind
	origRo, _ := c.Client.Scheme().New(obj.GetObjectKind().GroupVersionKind())
	orig := origRo.(client.Object)

	if err := c.Client.Get(ctx, client.ObjectKeyFromObject(obj), orig); err != nil && !apierrors.IsNotFound(err) {
		return false, err
	}

	// NOTE: remove this once this code is used for all "apply-style" functionality and has updated all the objects in the cluster.
	if isSsaMigrationNeeded(orig, c.NonSSAFieldOwner) {
		if err := c.MigrateToSSA(ctx, orig); err != nil {
			return false, err
		}
	}

	if err := c.Client.Patch(ctx, obj, client.Apply, client.FieldOwner(c.FieldOwner), client.ForceOwnership); err != nil {
		return false, fmt.Errorf("unable to patch '%s' called '%s' in namespace '%s': %w", obj.GetObjectKind().GroupVersionKind(), obj.GetName(), obj.GetNamespace(), err)
	}

	if config.determineUpdate {
		updated = obj.GetGeneration() != orig.GetGeneration()
	}

	return updated, nil
}

func prepareForSSA(obj client.Object, scheme *runtime.Scheme) error {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return err
	}
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	obj.SetManagedFields(nil)

	return nil
}

// Apply applies the objects, ie, creates or updates them on the cluster
// returns `true, nil` if at least one of the objects was created or modified,
// `false, nil` if nothing changed, and `false, err` if an error occurred
//
// NOTE: this is only used in tests
func (c *SsaApplyClient) Apply(ctx context.Context, toolchainObjects []client.Object, opts ...SsaApplyObjectOption) (bool, error) {
	createdOrUpdated := false
	for _, toolchainObject := range toolchainObjects {
		result, err := c.ApplyObject(ctx, toolchainObject, opts...)
		if err != nil {
			return createdOrUpdated, errors.Wrapf(err, "unable to create resource of kind: %s, version: %s", toolchainObject.GetObjectKind().GroupVersionKind().Kind, toolchainObject.GetObjectKind().GroupVersionKind().Version)
		}
		createdOrUpdated = createdOrUpdated || result
	}
	return createdOrUpdated, nil
}

func (c *SsaApplyClient) MigrateToSSA(ctx context.Context, obj client.Object) error {
	if err := csaupgrade.UpgradeManagedFields(obj, sets.New(c.NonSSAFieldOwner), c.FieldOwner); err != nil {
		return err
	}

	return c.Client.Update(ctx, obj)
}

func isSsaMigrationNeeded(obj client.Object, nonSsaOwner string) bool {
	for _, mf := range obj.GetManagedFields() {
		if mf.Manager == nonSsaOwner {
			return true
		}
	}

	return false
}
