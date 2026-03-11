package test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint: staticcheck // not deprecated anymore: see https://github.com/kubernetes-sigs/controller-runtime/pull/1101
)

// NewFakeClient creates a fake K8s client with ability to override specific Get/List/Create/Update/StatusUpdate/Delete functions
func NewFakeClient(t T, initObjs ...client.Object) *FakeClient {
	return newFakeClient(t, false, initObjs...)
}

// NewFakeClientWithManagedFields is like NewFakeClient but configures the fake client to return managed fields.
// This is needed for tests that verify SSA managed fields behavior.
func NewFakeClientWithManagedFields(t T, initObjs ...client.Object) *FakeClient {
	return newFakeClient(t, true, initObjs...)
}

func newFakeClient(t T, returnManagedFields bool, initObjs ...client.Object) *FakeClient {
	s := scheme.Scheme
	err := toolchainv1alpha1.AddToScheme(s)
	require.NoError(t, err)

	toolchainObjs := getAllToolchainResources(s)

	builder := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(initObjs...).
		WithStatusSubresource(toolchainObjs...)
	if returnManagedFields {
		builder = builder.WithReturnManagedFields()
	}
	cl := builder.Build()
	return &FakeClient{Client: cl, T: t}
}

type FakeClient struct {
	client.Client
	T                T
	MockGet          func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
	MockList         func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
	MockCreate       func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	MockUpdate       func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
	MockPatch        func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error
	MockStatusCreate func(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error
	MockStatusUpdate func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error
	MockStatusPatch  func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error
	MockDelete       func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
	MockDeleteAllOf  func(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error
}

type mockStatusUpdate struct {
	mockCreate func(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error
	mockUpdate func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error
	mockPatch  func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error
}

func (m *mockStatusUpdate) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return m.mockCreate(ctx, obj, subResource, opts...)
}

func (m *mockStatusUpdate) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return m.mockUpdate(ctx, obj, opts...)
}

func (m *mockStatusUpdate) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return m.mockPatch(ctx, obj, patch, opts...)
}

func (c *FakeClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if c.MockGet != nil {
		return c.MockGet(ctx, key, obj, opts...)
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

func (c *FakeClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if c.MockList != nil {
		return c.MockList(ctx, list, opts...)
	}
	return c.Client.List(ctx, list, opts...)
}

func (c *FakeClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if c.MockCreate != nil {
		return c.MockCreate(ctx, obj, opts...)
	}
	return Create(ctx, c, obj, opts...)
}

func Create(ctx context.Context, cl *FakeClient, obj client.Object, opts ...client.CreateOption) error {
	// Set Generation to `1` for newly created objects since the kube fake client doesn't set it
	obj.SetGeneration(1)
	return cl.Client.Create(ctx, obj, opts...)
}

func (c *FakeClient) Status() client.StatusWriter {
	m := mockStatusUpdate{}
	if c.MockStatusUpdate == nil && c.MockStatusPatch == nil && c.MockStatusCreate == nil {
		return c.Client.Status()
	}
	if c.MockStatusUpdate != nil {
		m.mockUpdate = c.MockStatusUpdate
	}
	if c.MockStatusPatch != nil {
		m.mockPatch = c.MockStatusPatch
	}
	if c.MockStatusCreate != nil {
		m.mockCreate = c.MockStatusCreate
	}
	return &m
}

func (c *FakeClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if c.MockUpdate != nil {
		return c.MockUpdate(ctx, obj, opts...)
	}
	return Update(ctx, c, obj, opts...)
}

func Update(ctx context.Context, cl *FakeClient, obj client.Object, opts ...client.UpdateOption) error {
	current, err := cleanObject(obj)
	if err != nil {
		return err
	}
	if err := cl.Client.Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, current); err != nil {
		return err
	}

	generationShouldChange, err := isGenerationChangeNeeded(current, obj)
	if err != nil {
		return err
	}

	if generationShouldChange {
		obj.SetGeneration(current.GetGeneration() + 1)
	} else {
		obj.SetGeneration(current.GetGeneration())
	}

	// Work around a controller-runtime fake client bug where Update unconditionally
	// replaces managed fields with the stored version, discarding caller-set values.
	// A real API server preserves managed fields set by the caller, which is needed
	// for csaupgrade.UpgradeManagedFields to work correctly.
	managedFields := obj.GetManagedFields()

	if err := cl.Client.Update(ctx, obj, opts...); err != nil {
		return err
	}

	if managedFields != nil {
		// Fix the stored managed fields in the tracker using a MergePatch.
		// The fake client's Update unconditionally replaces managed fields with
		// the stored version, so we use Patch to correct them.
		mfJSON, err := json.Marshal(map[string]any{
			"metadata": map[string]any{
				"managedFields": managedFields,
			},
		})
		if err != nil {
			return err
		}
		if err := cl.Client.Patch(ctx, obj, client.RawPatch(types.MergePatchType, mfJSON)); err != nil {
			return err
		}
		obj.SetManagedFields(managedFields)
	}

	return nil
}

func isGenerationChangeNeeded(currentObj, updatedObj client.Object) (bool, error) {
	// Update Generation if needed since the kube fake client doesn't update generations.
	// Increment the generation if spec (for objects with Spec) or data/stringData (for objects like CM and Secrets) is changed.
	updatingMap, err := toMap(updatedObj)
	if err != nil {
		return false, err
	}
	updatingMap["metadata"] = nil
	updatingMap["status"] = nil
	updatingMap["kind"] = nil
	updatingMap["apiVersion"] = nil
	if updatingMap["spec"] == nil {
		updatingMap["spec"] = map[string]interface{}{}
	}

	currentMap, err := toMap(currentObj)
	if err != nil {
		return false, err
	}
	currentMap["metadata"] = nil
	currentMap["status"] = nil
	currentMap["kind"] = nil
	currentMap["apiVersion"] = nil
	if currentMap["spec"] == nil {
		currentMap["spec"] = map[string]interface{}{}
	}
	for key, value := range currentMap {
		if _, exist := updatingMap[key]; !exist && value == nil {
			updatingMap[key] = nil
		}
	}

	return !reflect.DeepEqual(updatingMap, currentMap), nil
}

func cleanObject(obj client.Object) (client.Object, error) {
	newObj, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return nil, fmt.Errorf("unable cast the deepcopy of the object to client.Object: %+v", obj)
	}

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

func getAllToolchainResources(s *runtime.Scheme) []client.Object {
	kindToTypeMap := s.KnownTypes(toolchainv1alpha1.GroupVersion)
	toolchainObjs := make([]client.Object, 0, len(kindToTypeMap))
	for kind := range kindToTypeMap {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(toolchainv1alpha1.GroupVersion.WithKind(kind))
		toolchainObjs = append(toolchainObjs, obj)
	}
	return toolchainObjs
}

func (c *FakeClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if c.MockDelete != nil {
		return c.MockDelete(ctx, obj, opts...)
	}
	return c.Client.Delete(ctx, obj, opts...)
}

func (c *FakeClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	if c.MockDeleteAllOf != nil {
		return c.MockDeleteAllOf(ctx, obj, opts...)
	}
	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

func (c *FakeClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if c.MockPatch != nil {
		return c.MockPatch(ctx, obj, patch, opts...)
	}
	return Patch(ctx, c, obj, patch, opts...)
}

func Patch(ctx context.Context, fakeClient *FakeClient, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	// Our tests assume that an update to the spec increases the Generation - this is what we do by default in the Create and Update
	// methods, too. We need to replicate this behavior in Patch, too.

	// SSA patches are fully supported by the fake client now, so we just need to handle the generation bump.

	found := true
	orig := obj.DeepCopyObject().(client.Object)
	if err := fakeClient.Get(ctx, client.ObjectKeyFromObject(orig), orig); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		found = false
	}

	if found {
		// we need to figure out whether we should update the generation or not.
		// We do that by applying the patch in a dry-run and comparing the changes it made
		// to the original object.
		//
		// If the generation should change, we bump the generation of the object going into
		// the patch so that it is applied as such in the fake client.

		dryRunOpts := make([]client.PatchOption, len(opts)+1)
		copy(dryRunOpts, opts)
		dryRunOpts[len(opts)] = client.DryRunAll
		dryRunObj := obj.DeepCopyObject().(client.Object)
		if err := fakeClient.Client.Patch(ctx, dryRunObj, patch, dryRunOpts...); err != nil {
			return err
		}

		var err error
		shouldUpdateGeneration, err := isGenerationChangeNeeded(orig, dryRunObj)
		if err != nil {
			return err
		}

		if shouldUpdateGeneration {
			obj.SetGeneration(orig.GetGeneration() + 1)
		}
	} else if patch == client.Apply {
		// SSA Apply creates the object if it doesn't exist. Set Generation to 1
		// since the fake client doesn't do this automatically.
		obj.SetGeneration(1)
	}

	return fakeClient.Client.Patch(ctx, obj, patch, opts...)
}
