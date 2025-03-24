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
	s := scheme.Scheme
	err := toolchainv1alpha1.AddToScheme(s)
	require.NoError(t, err)

	toolchainObjs := getAllToolchainResources(s)

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(initObjs...).
		WithStatusSubresource(toolchainObjs...).
		Build()
	return &FakeClient{Client: cl, T: t}
}

func FakeSSA(fakeClient *FakeClient) {
	fakeClient.MockPatch = FakeSSAPatchFunction(fakeClient)
}

func FakeSSAPatchFunction(fakeClient *FakeClient) func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
		// fake client doesn't support SSA yet, so we have to be creative here and try to mock it out. Hopefully, SSA will be merged soon and we will
		// be able to remove this.
		//
		// NOTE: this doesn't really implement SSA in any sense. It is just here so that the existing tests pass.

		// A non-SSA patch assumes the object must already exist and should break if it doesn't. The SSA patch, on the other hand, creates the object
		// if it doesn't exist.

		found := true
		orig := obj.DeepCopyObject().(client.Object)
		if err := fakeClient.Get(ctx, client.ObjectKeyFromObject(orig), orig); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			found = false
		}

		if patch == client.Apply {
			if !found {
				if err := Create(ctx, fakeClient, obj); err != nil {
					return err
				}
			}
			// the fake client actively complains if it sees an SSA patch...
			patch = client.Merge
		}

		if found {
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
		}

		return fakeClient.Client.Patch(ctx, obj, patch, opts...)
	}
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

	return cl.Client.Update(ctx, obj, opts...)
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
	return c.Client.Patch(ctx, obj, patch, opts...)
}
