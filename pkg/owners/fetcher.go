package owners

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// OwnerFetcher fetches the owner references of Kubernetes objects by traversing
// the owner reference chain up to the top-level owner.
type OwnerFetcher struct {
	resourceLists   []*metav1.APIResourceList // All available API in the cluster
	discoveryClient discovery.ServerResourcesInterface
	dynamicClient   dynamic.Interface
}

// NewOwnerFetcher creates a new OwnerFetcher with the provided discovery and dynamic clients.
// The discovery client is used to fetch available API resources, and the dynamic client is used
// to retrieve owner objects from the cluster.
func NewOwnerFetcher(discoveryClient discovery.ServerResourcesInterface, dynamicClient dynamic.Interface) *OwnerFetcher {
	return &OwnerFetcher{
		discoveryClient: discoveryClient,
		dynamicClient:   dynamicClient,
	}
}

// ObjectWithGVR contains an unstructured Kubernetes object along with its
// GroupVersionResource (GVR) for identifying the resource type.
type ObjectWithGVR struct {
	Object *unstructured.Unstructured
	GVR    *schema.GroupVersionResource
}

// GetOwners recursively retrieves all owner references for the given object, starting from
// the immediate owner up to the top-level owner. It returns a slice of ObjectWithGVR in order
// from top-level owner to immediate owner. Returns nil if the object has no owner.
func (o *OwnerFetcher) GetOwners(ctx context.Context, obj metav1.Object) ([]*ObjectWithGVR, error) {
	if o.resourceLists == nil {
		// Get all API resources from the cluster using the discovery client. We need it for constructing GVRs for unstructured objects.
		// Do it here once, so we do not have to list it multiple times before listing/getting every unstructured resource.
		resourceLists, err := o.discoveryClient.ServerPreferredResources()
		if err != nil {
			return nil, err
		}
		o.resourceLists = resourceLists
	}

	// get the controller owner (it's possible to have only one controller owner)
	owners := obj.GetOwnerReferences()
	var ownerReference metav1.OwnerReference
	var nonControllerOwner metav1.OwnerReference
	for _, ownerRef := range owners {
		// try to get the controller owner as the preferred one
		if ownerRef.Controller != nil && *ownerRef.Controller {
			ownerReference = ownerRef
			break
		} else if nonControllerOwner.Name == "" {
			// take only the first non-controller owner
			nonControllerOwner = ownerRef
		}
	}
	// if no controller owner was found, then use the first non-controller owner (if present)
	if ownerReference.Name == "" {
		ownerReference = nonControllerOwner
	}
	if ownerReference.Name == "" {
		return nil, nil // No owner
	}
	// Get the GVR for the owner
	gvr, namespaced, err := gvrForKind(ownerReference.Kind, ownerReference.APIVersion, o.resourceLists)
	if err != nil {
		return nil, err
	}
	// Get the owner object; use namespace only for namespaced resources
	resourceClient := o.dynamicClient.Resource(*gvr)
	var ownerObject *unstructured.Unstructured
	if namespaced {
		ownerObject, err = resourceClient.Namespace(obj.GetNamespace()).Get(ctx, ownerReference.Name, metav1.GetOptions{})
	} else {
		ownerObject, err = resourceClient.Get(ctx, ownerReference.Name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch owner object %s %s (in namespace %s): %w", ownerReference.Name, gvr.String(), obj.GetNamespace(), err)
	}
	owner := &ObjectWithGVR{
		Object: ownerObject,
		GVR:    gvr,
	}
	// Recursively try to find the top owner
	ownerOwners, err := o.GetOwners(ctx, ownerObject)
	if err != nil || ownerOwners == nil {
		return append(ownerOwners, owner), err
	}
	return append(ownerOwners, owner), nil
}

// gvrForKind returns GVR and whether the resource is namespaced for the kind,
// if it's found in the available API list in the cluster.
// Returns an error if not found or failed to parse the API version.
func gvrForKind(kind, apiVersion string, resourceLists []*metav1.APIResourceList) (*schema.GroupVersionResource, bool, error) {
	gvr, namespaced, err := findGVRForKind(kind, apiVersion, resourceLists)
	if gvr == nil && err == nil {
		return nil, false, fmt.Errorf("no resource found for kind %s in %s", kind, apiVersion)
	}
	return gvr, namespaced, err
}

// findGVRForKind returns GVR and whether the resource is namespaced for the kind,
// if it's found in the available API list in the cluster.
// If not found then returns nil, false, nil.
// Returns nil, false, error if failed to parse the API version.
func findGVRForKind(kind, apiVersion string, resourceLists []*metav1.APIResourceList) (*schema.GroupVersionResource, bool, error) {
	// Parse the group and version from the APIVersion (e.g., "apps/v1" -> group: "apps", version: "v1")
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse APIVersion %s: %w", apiVersion, err)
	}

	// Look for a matching resource
	for _, resourceList := range resourceLists {
		if resourceList.GroupVersion == apiVersion {
			for _, apiResource := range resourceList.APIResources {
				if apiResource.Kind == kind {
					// Construct the GVR
					return &schema.GroupVersionResource{
						Group:    gv.Group,
						Version:  gv.Version,
						Resource: apiResource.Name,
					}, apiResource.Namespaced, nil
				}
			}
		}
	}

	return nil, false, nil
}
