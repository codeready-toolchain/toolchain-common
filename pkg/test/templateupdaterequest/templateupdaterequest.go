package templateupdaterequest

import (
	"fmt"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewTemplateUpdateRequest creates a specified number of TemplateRequestUpdate objects, with options
func NewTemplateUpdateRequest(name string, options ...Option) *toolchainv1alpha1.TemplateUpdateRequest {
	r := &toolchainv1alpha1.TemplateUpdateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.HostOperatorNs,
			Labels: map[string]string{
				toolchainv1alpha1.NSTemplateTierNameLabelKey: "basic",
			},
		},
	}
	for _, opt := range options {
		opt.applyToTemplateUpdateRequest(r)
	}
	return r
}

// NewTemplateUpdateRequests creates a specified number of TemplateRequestUpdate objects, with options
func NewTemplateUpdateRequests(size int, nameFmt string, options ...Option) []runtime.Object {
	templateUpdateRequests := make([]runtime.Object, size)
	for i := 0; i < size; i++ {
		templateUpdateRequests[i] = NewTemplateUpdateRequest(fmt.Sprintf(nameFmt, i), options...)
	}
	return templateUpdateRequests
}

// Option an option to configure a TemplateUpdateRequest
type Option interface {
	applyToTemplateUpdateRequest(*toolchainv1alpha1.TemplateUpdateRequest)
}

// DeletionTimestamp sets a deletion timestamp on the TemplateUpdateRequest with the given name (when creating a set of resources, the n-th may be marked for deletion)
type DeletionTimestamp string

var _ Option = DeletionTimestamp("")

func (d DeletionTimestamp) applyToTemplateUpdateRequest(r *toolchainv1alpha1.TemplateUpdateRequest) {
	if r.Name == string(d) {
		deletionTS := metav1.NewTime(time.Now())
		r.DeletionTimestamp = &deletionTS
	}
}

// TierName sets the name of the tier that was updated
type TierName string

var _ Option = TierName("")

func (t TierName) applyToTemplateUpdateRequest(r *toolchainv1alpha1.TemplateUpdateRequest) {
	r.Spec.TierName = string(t)
	r.Labels = map[string]string{
		toolchainv1alpha1.NSTemplateTierNameLabelKey: string(t),
	}
}
