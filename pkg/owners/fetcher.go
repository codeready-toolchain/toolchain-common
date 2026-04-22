package owners

import (
	"context"
	"fmt"

	"github.com/codeready-toolchain/toolchain-common/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// OwnerFetcher fetches the owner references of Kubernetes objects by traversing
// the owner reference chain up to the top-level owner.
type OwnerFetcher struct {
	resourceCache *client.ResourceCache
	dynamicClient dynamic.Interface
}

// NewOwnerFetcher creates a new OwnerFetcher with the provided discovery and dynamic clients.
// The discovery client is used to fetch available API resources, and the dynamic client is used
// to retrieve owner objects from the cluster.
// NOTE: this is kept for backwards compatibility. Prefer using the NewOwnerFetcherWithCache() function.
func NewOwnerFetcher(discoveryClient discovery.ServerResourcesInterface, dynamicClient dynamic.Interface) *OwnerFetcher {
	return NewOwnerFetcherWithCache(dynamicClient, client.NewResourceCache(discoveryClient))
}

// NewOwnerFetcherWithCache creates a new OwnerFetcher with the provided resourceCache used to look up
// the GVRs and the dynamicClient for retrieval of the owners from the cluster.
func NewOwnerFetcherWithCache(dynamicClient dynamic.Interface, resourceCache *client.ResourceCache) *OwnerFetcher {
	return &OwnerFetcher{
		resourceCache: resourceCache,
		dynamicClient: dynamicClient,
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
	gvr, found, namespaced, err := o.resourceCache.GVRForKind(ownerReference.Kind, ownerReference.APIVersion)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no resource found for kind %s in %s", ownerReference.Kind, ownerReference.APIVersion)
	}

	// Get the owner object; use namespace only for namespaced resources
	resourceClient := o.dynamicClient.Resource(gvr)
	var ownerObject *unstructured.Unstructured
	nsdName := ownerReference.Name
	if namespaced {
		ownerObject, err = resourceClient.Namespace(obj.GetNamespace()).Get(ctx, ownerReference.Name, metav1.GetOptions{})
		nsdName = fmt.Sprintf("%s/%s", obj.GetNamespace(), ownerReference.Name)
	} else {
		ownerObject, err = resourceClient.Get(ctx, ownerReference.Name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch owner object %s %s : %w", nsdName, gvr.String(), err)
	}
	owner := &ObjectWithGVR{
		Object: ownerObject,
		GVR:    &gvr,
	}
	// Recursively try to find the top owner
	ownerOwners, err := o.GetOwners(ctx, ownerObject)
	if err != nil || ownerOwners == nil {
		return append(ownerOwners, owner), err
	}
	return append(ownerOwners, owner), nil
}
