package idler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/codeready-toolchain/toolchain-common/pkg/owners"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SupportedScaleResources maps Camel kinds that must be idled via the scale subresource
// (rather than a direct dynamic client patch of spec.replicas).
var SupportedScaleResources = map[schema.GroupVersionKind]schema.GroupVersionResource{
	schema.GroupVersion{Group: "camel.apache.org", Version: "v1"}.WithKind("Integration"):          schema.GroupVersion{Group: "camel.apache.org", Version: "v1"}.WithResource("integrations"),
	schema.GroupVersion{Group: "camel.apache.org", Version: "v1alpha1"}.WithKind("KameletBinding"): schema.GroupVersion{Group: "camel.apache.org", Version: "v1alpha1"}.WithResource("kameletbindings"),
}

func (i *Idler) scaleToZero(ctx context.Context, objectWithGVR *owners.ObjectWithGVR) error {
	object := objectWithGVR.Object
	logger := log.FromContext(ctx).WithValues("kind", object.GetObjectKind().GroupVersionKind().Kind, "name", object.GetName())
	logger.Info("Scaling controller owner to zero")

	patch := []byte(`{"spec":{"replicas":0}}`)
	if scaleGVR, ok := SupportedScaleResources[object.GetObjectKind().GroupVersionKind()]; ok {
		logger.Info("Scaling controller owner to zero using the scale subresource")
		_, err := i.scalesClient.Scales(object.GetNamespace()).Patch(ctx, scaleGVR, object.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
		if err != nil {
			return err
		}
		logger.Info("Controller owner scaled to zero using the scale subresource")
		return nil
	}

	_, err := i.dynamicClient.
		Resource(*objectWithGVR.GVR).
		Namespace(object.GetNamespace()).
		Patch(ctx, object.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	logger.Info("Controller owner scaled to zero")
	return nil
}

// idleAAP idles AAP instance if not already idled
func (i *Idler) idleAAP(ctx context.Context, objectWithGVR *owners.ObjectWithGVR) error {
	return i.idleBySpecBool(ctx, objectWithGVR, "idle_aap", "AAP")
}

// idleClaw idles a Claw instance if not already idled
func (i *Idler) idleClaw(ctx context.Context, objectWithGVR *owners.ObjectWithGVR) error {
	return i.idleBySpecBool(ctx, objectWithGVR, "idle", "Claw")
}

// idleBySpecBool sets the given spec bool field to true when the resource is not already idled.
func (i *Idler) idleBySpecBool(ctx context.Context, objectWithGVR *owners.ObjectWithGVR, field, label string) error {
	name := objectWithGVR.Object.GetName()
	logger := log.FromContext(ctx).WithValues("name", name)
	idled, _, err := unstructured.NestedBool(objectWithGVR.Object.UnstructuredContent(), "spec", field)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to parse %s CR to get the spec.%s field", label, field))
	}
	if idled {
		logger.Info(fmt.Sprintf("%s CR is already idled", label))
		return nil
	}
	logger.Info(fmt.Sprintf("Idling %s", label))

	patch := fmt.Appendf(nil, `{"spec":{%q:true}}`, field)
	_, err = i.dynamicClient.
		Resource(*objectWithGVR.GVR).
		Namespace(objectWithGVR.Object.GetNamespace()).
		Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("%s idled", label), "name", name)
	return nil
}

func (i *Idler) deleteResource(ctx context.Context, objectWithGVR *owners.ObjectWithGVR) error {
	logger := log.FromContext(ctx)
	object := objectWithGVR.Object
	logger.Info("Deleting controller owner",
		"kind", object.GetObjectKind().GroupVersionKind().Kind, "name", object.GetName())
	// see https://github.com/kubernetes/kubernetes/issues/20902#issuecomment-321484735
	// also, this may be needed for the e2e tests if the call to `client.Delete` comes too quickly after creating the job,
	// which may leave the job's pod running but orphan, hence causing a test failure (and making the test flaky)
	propagationPolicy := metav1.DeletePropagationBackground

	err := i.dynamicClient.
		Resource(*objectWithGVR.GVR).
		Namespace(object.GetNamespace()).
		Delete(ctx, object.GetName(), metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	if err != nil {
		return err
	}

	logger.Info("Controller owner deleted",
		"kind", object.GetObjectKind().GroupVersionKind().Kind, "name", object.GetName())
	return nil
}

func (i *Idler) scaleDeploymentConfigToZero(ctx context.Context, objectWithGVR *owners.ObjectWithGVR) error {
	logger := log.FromContext(ctx)
	object := objectWithGVR.Object
	logger.Info("Scaling DeploymentConfig to zero", "name", object.GetName())
	patch := []byte(`{"spec":{"replicas":0,"paused":false}}`)
	_, err := i.dynamicClient.
		Resource(*objectWithGVR.GVR).
		Namespace(object.GetNamespace()).
		Patch(ctx, object.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	log.FromContext(ctx).Info("DeploymentConfig scaled to zero", "name", object.GetName())
	return nil
}

func (i *Idler) stopVirtualMachine(ctx context.Context, objectWithGVR *owners.ObjectWithGVR) error {
	logger := log.FromContext(ctx)
	object := objectWithGVR.Object
	logger.Info("Stopping VirtualMachine", "name", object.GetName())
	err := i.restClient.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, "v1")).
		Namespace(object.GetNamespace()).
		Resource("virtualmachines").
		Name(object.GetName()).
		SubResource("stop").
		Do(ctx).
		Error()
	if err != nil {
		return err
	}

	logger.Info("VirtualMachine stopped", "name", object.GetName())
	return nil
}

// idleServingRuntime idles ServingRuntime by deleting InferenceService objects that exist for longer than the timeout.
// timeoutSeconds <= 0 deletes nothing (on-demand callers pass 0 when no cutoff applies).
func (i *Idler) idleServingRuntime(ctx context.Context, objectWithGVR *owners.ObjectWithGVR, timeoutSeconds int32) error {
	logger := log.FromContext(ctx)
	namespace := objectWithGVR.Object.GetNamespace()

	logger.Info("Idling ServingRuntime by deleting old InferenceService objects", "name", objectWithGVR.Object.GetName())

	if timeoutSeconds <= 0 {
		logger.Info("Skipping InferenceService cleanup because timeoutSeconds is not positive", "timeoutSeconds", timeoutSeconds)
		return nil
	}

	inferenceServiceGVR := schema.GroupVersionResource{
		Group:    "serving.kserve.io",
		Version:  "v1beta1",
		Resource: "inferenceservices",
	}

	inferenceServiceList, err := i.dynamicClient.
		Resource(inferenceServiceGVR).
		Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list InferenceService objects: %w", err)
	}

	cutoffTime := time.Now().Add(-time.Duration(timeoutSeconds) * time.Second)
	var deletionErrors []error

	for _, inferenceService := range inferenceServiceList.Items {
		creationTime := inferenceService.GetCreationTimestamp().Time
		if creationTime.Before(cutoffTime) {
			logger.Info("Deleting old InferenceService", "name", inferenceService.GetName(), "age", time.Since(creationTime))

			err := i.dynamicClient.
				Resource(inferenceServiceGVR).
				Namespace(namespace).
				Delete(ctx, inferenceService.GetName(), metav1.DeleteOptions{})
			if err != nil {
				deletionErrors = append(deletionErrors, err)
			} else {
				logger.Info("InferenceService deleted", "name", inferenceService.GetName())
			}
		}
	}

	return errors.Join(deletionErrors...)
}
