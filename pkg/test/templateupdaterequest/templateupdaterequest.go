package templateupdaterequest

import (
	"fmt"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewTemplateUpdateRequests creates a specified number of TemplateRequestUpdate objects, with options
func NewTemplateUpdateRequests(size int, options ...Option) []runtime.Object {
	templateUpdateRequests := make([]runtime.Object, size)
	for i := 0; i < size; i++ {
		r := &toolchainv1alpha1.TemplateUpdateRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("user-%d", i),
				Namespace: test.HostOperatorNs,
				Labels: map[string]string{
					toolchainv1alpha1.NSTemplateTierNameLabelKey: "basic",
				},
			},
		}
		for _, opt := range options {
			opt.applyToTemplateUpdateRequest(i, r)
		}
		templateUpdateRequests[i] = r
	}
	return templateUpdateRequests
}

// Option an option to configure a TemplateUpdateRequest
type Option interface {
	applyToTemplateUpdateRequest(int, *toolchainv1alpha1.TemplateUpdateRequest)
}

// NameFormat defines the format of the resource name
type NameFormat string

var _ Option = NameFormat("")

func (f NameFormat) applyToTemplateUpdateRequest(i int, mur *toolchainv1alpha1.TemplateUpdateRequest) {
	mur.ObjectMeta.Name = fmt.Sprintf(string(f), i)
}

// DeletionTimestamp sets a deletion timestamp on the TemplateUpdateRequest with the given index (when creating a set of resources, the n-th may be marked for deletion)
type DeletionTimestamp int

var _ Option = DeletionTimestamp(0)

func (d DeletionTimestamp) applyToTemplateUpdateRequest(i int, r *toolchainv1alpha1.TemplateUpdateRequest) {
	if i == int(d) {
		deletionTS := metav1.NewTime(time.Now())
		r.DeletionTimestamp = &deletionTS
	}
}

// TierName sets the name of the tier that was updated
type TierName string

var _ Option = TierName("")

func (t TierName) applyToTemplateUpdateRequest(_ int, r *toolchainv1alpha1.TemplateUpdateRequest) {
	r.Spec.TierName = string(t)
	r.Labels = map[string]string{
		toolchainv1alpha1.NSTemplateTierNameLabelKey: string(t),
	}
}
