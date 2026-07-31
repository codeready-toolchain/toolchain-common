package idler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/codeready-toolchain/toolchain-common/pkg/owners"
	"github.com/go-logr/logr"
	"github.com/redhat-cop/operator-utils/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/scale"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const vmSubresourceURLFmt = "/apis/subresources.kubevirt.io/%s"

// ErrUnsupportedKind is returned by IdleOwner when the owner kind is not in the idle matrix.
var ErrUnsupportedKind = errors.New("unsupported idle owner kind")

// Idler applies owner-idle actions (scale/stop/patch/delete) for known workload kinds.
type Idler struct {
	ownerFetcher  *owners.OwnerFetcher
	dynamicClient dynamic.Interface
	scalesClient  scale.ScalesGetter
	restClient    rest.Interface
}

// Options configures idle actions that need caller-supplied parameters.
type Options struct {
	// TimeoutSeconds is the ServingRuntime InferenceService age cutoff.
	// InferenceServices older than now-TimeoutSeconds are deleted.
	// 0 means delete nothing.
	TimeoutSeconds int32
}

// New creates an Idler with the clients required for the full owner-idle kind matrix.
func New(discoveryClient discovery.ServerResourcesInterface, dynamicClient dynamic.Interface, scalesClient scale.ScalesGetter, restClient rest.Interface) *Idler {
	return &Idler{
		ownerFetcher:  owners.NewOwnerFetcher(discoveryClient, dynamicClient),
		dynamicClient: dynamicClient,
		scalesClient:  scalesClient,
		restClient:    restClient,
	}
}

// IdleOwner applies the kind-specific idle action for a single known owner.
// Unknown kinds return ErrUnsupportedKind so callers can skip them.
func (i *Idler) IdleOwner(ctx context.Context, ownerWithGVR *owners.ObjectWithGVR, opts Options) error {
	ownerKind := ownerWithGVR.Object.GetObjectKind().GroupVersionKind().Kind
	switch ownerKind {
	case "Deployment", "ReplicaSet", "Integration", "KameletBinding", "StatefulSet", "ReplicationController":
		return i.scaleToZero(ctx, ownerWithGVR)
	case "DaemonSet", "Job", "DataVolume", "PersistentVolumeClaim":
		return i.deleteResource(ctx, ownerWithGVR)
	case "DeploymentConfig":
		return i.scaleDeploymentConfigToZero(ctx, ownerWithGVR)
	case "VirtualMachine":
		return i.stopVirtualMachine(ctx, ownerWithGVR)
	case "AnsibleAutomationPlatform":
		return i.idleAAP(ctx, ownerWithGVR)
	case "Claw":
		return i.idleClaw(ctx, ownerWithGVR)
	case "ServingRuntime":
		return i.idleServingRuntime(ctx, ownerWithGVR, opts.TimeoutSeconds)
	default:
		return ErrUnsupportedKind
	}
}

// IdleFromPod walks the pod's owner chain and idles up to two known owners.
// Unlike the member-operator timeout path, this always tries the second known owner
// when present (no 105%/110% timeout percentages). It never creates Notifications.
// Returns the kind and name of the first known owner that was attempted.
func (i *Idler) IdleFromPod(ctx context.Context, pod *corev1.Pod, opts Options) (string, string, error) {
	logger := log.FromContext(ctx)
	logger.Info("Idling owners from pod")

	ownerChain, err := i.ownerFetcher.GetOwners(ctx, pod)
	if err != nil {
		logger.Error(err, "failed to find all owners, try to idle the workload with information that is available")
	}

	logOwnershipChain(logger, ownerChain, pod)

	var topOwnerKind, topOwnerName string
	var errToReturn error
	for _, ownerWithGVR := range ownerChain {
		if util.IsBeingDeleted(ownerWithGVR.Object) {
			continue
		}
		owner := ownerWithGVR.Object
		ownerKind := owner.GetObjectKind().GroupVersionKind().Kind

		err = i.IdleOwner(ctx, ownerWithGVR, opts)
		if errors.Is(err, ErrUnsupportedKind) {
			continue
		}

		if topOwnerKind == "" {
			topOwnerKind = ownerKind
			topOwnerName = owner.GetName()
			errToReturn = err
		} else {
			errToReturn = errors.Join(errToReturn, err)
			break
		}
	}

	return topOwnerKind, topOwnerName, errToReturn
}

func logOwnershipChain(logger logr.Logger, ownerChain []*owners.ObjectWithGVR, pod *corev1.Pod) {
	if len(ownerChain) == 0 {
		logger.Info("No ownership chain, it's a standalone pod")
		return
	}
	chain := make([]string, 0, len(ownerChain)+1)
	for _, o := range ownerChain {
		gvk := o.Object.GetObjectKind().GroupVersionKind()
		if gvk.Group != "" {
			chain = append(chain, fmt.Sprintf("%s/%s.%s", o.Object.GetName(), gvk.Kind, gvk.Group))
		} else {
			chain = append(chain, fmt.Sprintf("%s/%s", o.Object.GetName(), gvk.Kind))
		}
	}
	chain = append(chain, fmt.Sprintf("%s/Pod", pod.Name))
	logger.Info("Ownership chain", "chain", strings.Join(chain, " -> "))
}
