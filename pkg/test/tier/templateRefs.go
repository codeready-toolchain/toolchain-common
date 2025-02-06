package tier

import (
	"encoding/json"
	"sort"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/hash"
)

// ComputeTemplateRefsHash computes the hash of the value of `status.revisions[]`
func ComputeTemplateRefsHash(tier *toolchainv1alpha1.NSTemplateTier) (string, error) {
	var refs []string
	for _, rev := range tier.Status.Revisions {
		refs = append(refs, rev)
	}
	sort.Strings(refs)
	m, err := json.Marshal(templateRefs{Refs: refs})
	if err != nil {
		return "", err
	}
	return hash.Encode(m), nil
}

// TemplateTierHashLabel returns the label key to specify the version of the templates of the given tier
func TemplateTierHashLabelKey(tierName string) string {
	return toolchainv1alpha1.LabelKeyPrefix + tierName + "-tier-hash"
}

type templateRefs struct {
	Refs []string `json:"refs"`
}
