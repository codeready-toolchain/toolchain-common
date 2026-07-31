package idler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

const (
	vmSubresourceURLFmt = "/apis/subresources.kubevirt.io/%s"

	// secondOwnerTimeoutRatio is the fraction of PodTimeoutSeconds after which
	// SecondOwnerAfterTimeout also tries a second known owner (even on success).
	secondOwnerTimeoutRatio = 1.05
	// podDeleteTimeoutRatio is the fraction of PodTimeoutSeconds after which
	// SecondOwnerAfterTimeout clears the owner and recommends pod deletion when
	// nothing remains attempted.
	podDeleteTimeoutRatio = 1.10
)

// ErrUnsupportedKind is returned by IdleOwner when the owner kind is not in the idle matrix,
// or when the kind requires deletion and SkipDeleteKinds is true.
var ErrUnsupportedKind = errors.New("unsupported idle owner kind")

// SecondOwnerPolicy controls when IdleFromPod idles a second known owner.
type SecondOwnerPolicy int

const (
	// SecondOwnerAlways tries up to two known owners (on-demand).
	SecondOwnerAlways SecondOwnerPolicy = iota
	// SecondOwnerAfterTimeout tries the second owner only when the first fails or the pod
	// has been running longer than 105% of PodTimeoutSeconds. When nothing remains
	// attempted and the pod exceeds 110%, Kind/Name are cleared and PodDeleteRecommended
	// is set so the caller may delete the pod.
	SecondOwnerAfterTimeout
)

// Idler applies owner-idle actions (scale/stop/patch/delete) for known workload kinds.
type Idler struct {
	ownerFetcher  *owners.OwnerFetcher
	dynamicClient dynamic.Interface
	scalesClient  scale.ScalesGetter
	restClient    rest.Interface
}

// Options configures idle actions and IdleFromPod owner-walk policy.
type Options struct {
	// TimeoutSeconds is the ServingRuntime InferenceService age cutoff.
	// InferenceServices older than now-TimeoutSeconds are deleted.
	// 0 means delete nothing.
	TimeoutSeconds int32

	// SkipDeleteKinds skips idling via delete for DaemonSet, Job, DataVolume,
	// PersistentVolumeClaim, and ServingRuntime (treated as unsupported).
	// The zero value (false) keeps the full matrix, including deletes.
	// On-demand callers should set this to true to skip deletes for the safer default.
	SkipDeleteKinds bool

	// SecondOwnerPolicy controls when a second known owner is idled.
	// The zero value is SecondOwnerAlways.
	SecondOwnerPolicy SecondOwnerPolicy

	// PodTimeoutSeconds is used with SecondOwnerAfterTimeout for the 105%/110% gates.
	// Ignored when SecondOwnerPolicy is SecondOwnerAlways.
	PodTimeoutSeconds int32
}

// Result is the outcome of IdleFromPod. Common never deletes the Pod; callers that
// allow last-resort pod deletion should act when PodDeleteRecommended is true.
type Result struct {
	Kind string
	Name string
	// PodDeleteRecommended is true when no known owner remains as the idle target
	// (standalone/unknown chain, or SecondOwnerAfterTimeout 110% stuck case).
	PodDeleteRecommended bool
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

// OwnerFetcher returns the OwnerFetcher used to walk controller owner chains.
func (i *Idler) OwnerFetcher() *owners.OwnerFetcher {
	return i.ownerFetcher
}

// IdleOwner applies the kind-specific idle action for a single known owner.
// Unknown kinds (and delete-kinds when SkipDeleteKinds is true) return ErrUnsupportedKind
// so callers can skip them.
func (i *Idler) IdleOwner(ctx context.Context, ownerWithGVR *owners.ObjectWithGVR, opts Options) error {
	ownerKind := ownerWithGVR.Object.GetObjectKind().GroupVersionKind().Kind
	switch ownerKind {
	case "Deployment", "ReplicaSet", "Integration", "KameletBinding", "StatefulSet", "ReplicationController":
		return i.scaleToZero(ctx, ownerWithGVR)
	case "DaemonSet", "Job", "DataVolume", "PersistentVolumeClaim":
		if opts.SkipDeleteKinds {
			return ErrUnsupportedKind
		}
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
		if opts.SkipDeleteKinds {
			return ErrUnsupportedKind
		}
		return i.idleServingRuntime(ctx, ownerWithGVR, opts.TimeoutSeconds)
	default:
		return ErrUnsupportedKind
	}
}

// IdleFromPod walks the pod's owner chain and idles up to two known owners.
// It never creates Notifications and never deletes the Pod; see Result.PodDeleteRecommended.
func (i *Idler) IdleFromPod(ctx context.Context, pod *corev1.Pod, opts Options) (Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Idling owners from pod")

	ownerChain, fetchErr := i.ownerFetcher.GetOwners(ctx, pod)
	if fetchErr != nil {
		logger.Error(fetchErr, "failed to find all owners, try to idle the workload with information that is available")
	}

	LogOwnershipChain(logger, ownerChain, pod)

	var topOwnerKind, topOwnerName string
	attempted := false
	var errToReturn error
	for _, ownerWithGVR := range ownerChain {
		if util.IsBeingDeleted(ownerWithGVR.Object) {
			if opts.SecondOwnerPolicy == SecondOwnerAfterTimeout {
				// Match member Idler: a deleting owner means the previous attempt did not stick.
				attempted = false
			}
			continue
		}

		err := i.IdleOwner(ctx, ownerWithGVR, opts)
		if errors.Is(err, ErrUnsupportedKind) {
			continue
		}

		owner := ownerWithGVR.Object
		ownerKind := owner.GetObjectKind().GroupVersionKind().Kind
		attempted = true
		if topOwnerKind == "" {
			topOwnerKind = ownerKind
			topOwnerName = owner.GetName()
			errToReturn = err
		} else {
			errToReturn = errors.Join(errToReturn, err)
			break
		}

		if opts.SecondOwnerPolicy == SecondOwnerAfterTimeout && !shouldTryNextOwner(pod, opts.PodTimeoutSeconds, err) {
			return resultForOwners(topOwnerKind, topOwnerName), nil
		}
		if opts.SecondOwnerPolicy == SecondOwnerAfterTimeout {
			logger.Info("Scaling the first known owner down either failed or the pod has been running for longer than 105% of the idler timeout. Scaling the next known owner.")
		}
		// SecondOwnerAlways (and AfterTimeout when continuing): try a second known owner when present.
	}

	if opts.SecondOwnerPolicy == SecondOwnerAfterTimeout && shouldRecommendPodDelete(pod, opts.PodTimeoutSeconds, attempted) {
		// Nothing remains attempted (e.g. top owner idled but remaining owners are deleting).
		// Clear owner so the caller can delete the pod as a last resort.
		return Result{PodDeleteRecommended: true}, errToReturn
	}

	if topOwnerKind == "" && fetchErr != nil {
		return Result{}, fetchErr
	}
	return resultForOwners(topOwnerKind, topOwnerName), errToReturn
}

// shouldTryNextOwner reports whether SecondOwnerAfterTimeout should continue after the first
// known owner. Stop when that idle succeeded and the pod is still under 105% of timeout.
func shouldTryNextOwner(pod *corev1.Pod, timeoutSeconds int32, idleErr error) bool {
	return idleErr != nil || podRunningLongerThan(pod, timeoutSeconds, secondOwnerTimeoutRatio)
}

// shouldRecommendPodDelete reports whether SecondOwnerAfterTimeout should clear the owner
// and recommend pod deletion (nothing remains attempted and the pod exceeds 110% of timeout).
func shouldRecommendPodDelete(pod *corev1.Pod, timeoutSeconds int32, attempted bool) bool {
	return !attempted && podRunningLongerThan(pod, timeoutSeconds, podDeleteTimeoutRatio)
}

func resultForOwners(kind, name string) Result {
	return Result{
		Kind:                 kind,
		Name:                 name,
		PodDeleteRecommended: kind == "",
	}
}

func podRunningLongerThan(pod *corev1.Pod, timeoutSeconds int32, ratio float64) bool {
	if pod.Status.StartTime == nil || timeoutSeconds <= 0 {
		return false
	}
	deadline := pod.Status.StartTime.Add(time.Duration(float64(timeoutSeconds)*ratio) * time.Second)
	return time.Now().After(deadline)
}

// LogOwnershipChain logs the controller ownership chain for the given pod.
func LogOwnershipChain(logger logr.Logger, ownerChain []*owners.ObjectWithGVR, pod *corev1.Pod) {
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
