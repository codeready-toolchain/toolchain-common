package client

import (
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

type ResourceCache struct {
	mutex           sync.Mutex                // guard the initialization ofthe resourceLists
	resourceLists   []*metav1.APIResourceList // All available API in the cluster
	discoveryClient discovery.ServerResourcesInterface
}

// NewResourceCache creates a new ResourceCache  with the provided discovery client.
// The discovery client is used to fetch available API resources.
func NewResourceCache(discoveryClient discovery.ServerResourcesInterface) *ResourceCache {
	return &ResourceCache{
		discoveryClient: discoveryClient,
	}
}

// GVRForKind returns a group-resource-version for the supplied kind and api version.
func (rc *ResourceCache) GVRForKind(kind, apiVersion string) (gvr schema.GroupVersionResource, found bool, namespaced bool, err error) {
	if err = rc.ensureResourceList(); err != nil {
		return
	}

	// Parse the group and version from the APIVersion (e.g., "apps/v1" -> group: "apps", version: "v1")
	var gv schema.GroupVersion
	gv, err = schema.ParseGroupVersion(apiVersion)
	if err != nil {
		err = fmt.Errorf("failed to parse APIVersion %s: %w", apiVersion, err)
		return
	}

	// Look for a matching resource
	for _, resourceList := range rc.resourceLists {
		if resourceList.GroupVersion == apiVersion {
			for _, apiResource := range resourceList.APIResources {
				if apiResource.Kind == kind {
					// Construct the GVR
					found = true
					gvr = schema.GroupVersionResource{
						Group:    gv.Group,
						Version:  gv.Version,
						Resource: apiResource.Name,
					}
					namespaced = apiResource.Namespaced

					return
				}
			}
		}
	}

	return
}

// GVKForGR given the group-resource, returns the first matching GVK for it.
func (rc *ResourceCache) GVKForGR(gr schema.GroupResource) (gvk schema.GroupVersionKind, found bool, err error) {
	if err = rc.ensureResourceList(); err != nil {
		return
	}

	for _, resourceList := range rc.resourceLists {
		var gv schema.GroupVersion
		gv, err = schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			err = fmt.Errorf("failed to parse GroupVersion %s: %w", resourceList.GroupVersion, err)
			return
		}
		if gv.Group != gr.Group {
			continue
		}
		for _, res := range resourceList.APIResources {
			if res.Name == gr.Resource {
				gvk = schema.GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: res.Kind}
				found = true
				return
			}
		}
	}

	return
}

func (rc *ResourceCache) ensureResourceList() error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	if rc.resourceLists == nil {
		// Get all API resources from the cluster using the discovery client. We need it for constructing GVRs for unstructured objects.
		// Do it here once, so we do not have to list it multiple times before listing/getting every unstructured resource.
		//
		// The ServerPreferredResources() method is meant to return partial results on failure.
		// We ignore them here for the sake of simplicity. Let's just retry to get the full results the next time.
		resourceList, err := rc.discoveryClient.ServerPreferredResources()
		if err != nil {
			return err
		}

		rc.resourceLists = resourceList
	}

	return nil
}
