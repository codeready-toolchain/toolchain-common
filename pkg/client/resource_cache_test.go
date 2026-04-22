package client

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type fakeDiscoveryClient struct {
	resources []*metav1.APIResourceList
	err       error
	calls     int
}

func (f *fakeDiscoveryClient) ServerResourcesForGroupVersion(string) (*metav1.APIResourceList, error) {
	return nil, nil
}

func (f *fakeDiscoveryClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return nil, nil, nil
}

func (f *fakeDiscoveryClient) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	f.calls++
	return f.resources, f.err
}

func (f *fakeDiscoveryClient) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return nil, nil
}

func TestGVRForKind(t *testing.T) {
	resources := []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Kind: "Pod", Namespaced: true},
				{Name: "nodes", Kind: "Node", Namespaced: false},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Name: "deployments", Kind: "Deployment", Namespaced: true},
				{Name: "daemonsets", Kind: "DaemonSet", Namespaced: true},
			},
		},
	}

	t.Run("finds namespaced core resource", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		gvr, found, namespaced, err := rc.GVRForKind("Pod", "v1")

		require.NoError(t, err)
		assert.True(t, found)
		assert.True(t, namespaced)
		assert.Equal(t, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, gvr)
	})

	t.Run("finds cluster-scoped core resource", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		gvr, found, namespaced, err := rc.GVRForKind("Node", "v1")

		require.NoError(t, err)
		assert.True(t, found)
		assert.False(t, namespaced)
		assert.Equal(t, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}, gvr)
	})

	t.Run("finds resource in non-core group", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		gvr, found, namespaced, err := rc.GVRForKind("Deployment", "apps/v1")

		require.NoError(t, err)
		assert.True(t, found)
		assert.True(t, namespaced)
		assert.Equal(t, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, gvr)
	})

	t.Run("not found", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		_, found, _, err := rc.GVRForKind("StatefulSet", "apps/v1")

		require.NoError(t, err)
		assert.False(t, found)
		assert.NoError(t, err)
	})

	t.Run("not found in wrong api version", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		_, found, _, err := rc.GVRForKind("Deployment", "apps/v1beta1")

		require.NoError(t, err)
		assert.False(t, found)
		assert.NoError(t, err)
	})

	t.Run("invalid api version", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		_, _, _, err := rc.GVRForKind("Pod", "a/b/c")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse APIVersion")
	})

	t.Run("discovery error", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{err: errors.New("discovery failed")})

		_, _, _, err := rc.GVRForKind("Pod", "v1")

		require.EqualError(t, err, "discovery failed")
	})

	t.Run("caches discovery results", func(t *testing.T) {
		dc := &fakeDiscoveryClient{resources: resources}
		rc := NewResourceCache(dc)

		_, _, _, err := rc.GVRForKind("Pod", "v1")
		require.NoError(t, err)

		_, _, _, err = rc.GVRForKind("Deployment", "apps/v1")
		require.NoError(t, err)

		assert.Equal(t, 1, dc.calls, "discovery should have been called only once")
	})
}

func TestGVKForGR(t *testing.T) {
	resources := []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Kind: "Pod", Namespaced: true},
				{Name: "services", Kind: "Service", Namespaced: true},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Name: "deployments", Kind: "Deployment", Namespaced: true},
				{Name: "replicasets", Kind: "ReplicaSet", Namespaced: true},
			},
		},
		{
			GroupVersion: "kubevirt.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "virtualmachines", Kind: "VirtualMachine", Namespaced: true},
			},
		},
	}

	t.Run("finds core group resource", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		gvk, found, err := rc.GVKForGR(schema.GroupResource{Group: "", Resource: "pods"})

		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}, gvk)
	})

	t.Run("finds non-core group resource", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		gvk, found, err := rc.GVKForGR(schema.GroupResource{Group: "apps", Resource: "deployments"})

		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}, gvk)
	})

	t.Run("finds resource with dotted group", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		gvk, found, err := rc.GVKForGR(schema.GroupResource{Group: "kubevirt.io", Resource: "virtualmachines"})

		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, schema.GroupVersionKind{Group: "kubevirt.io", Version: "v1", Kind: "VirtualMachine"}, gvk)
	})

	t.Run("not found", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		_, found, err := rc.GVKForGR(schema.GroupResource{Group: "apps", Resource: "statefulsets"})

		require.NoError(t, err)
		assert.False(t, found)
		assert.NoError(t, err)
	})

	t.Run("wrong group", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{resources: resources})

		_, found, err := rc.GVKForGR(schema.GroupResource{Group: "extensions", Resource: "deployments"})

		require.NoError(t, err)
		assert.False(t, found)
		assert.NoError(t, err)
	})

	t.Run("discovery error", func(t *testing.T) {
		rc := NewResourceCache(&fakeDiscoveryClient{err: errors.New("discovery failed")})

		_, _, err := rc.GVKForGR(schema.GroupResource{Group: "apps", Resource: "deployments"})

		require.EqualError(t, err, "discovery failed")
	})
}

