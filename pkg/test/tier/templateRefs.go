package tier

import (
	"encoding/json"
	"sort"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/hash"
)

// ComputeTemplateRefsHash computes the hash of the value of `status.revisions[]`
// currently it passes the `.spec.namespaces[].templateRef` in key field and `.spec.clusteResource.TemplateRef` value in Key field.
// as we want to make sure that we just use the values of `.spec.namespaces[].templateRef`+ `.spec.clusteResource.TemplateRef`
// and we do not use the extra values available in `status.revisions[]` since there is no logic yet to delete the extra key-value pairs
// if the extra values are used while calculating hash, it won't be equal to the hash of `NSTemplateSetSpec`
// TODO : once there is logic to have the `status.revisions[]` cleaned up, update this function to just loop over `status.revisions[]`
// to calculate hash
func ComputeTemplateRefsHash(tier *toolchainv1alpha1.NSTemplateTier) (string, error) {
	var refs []string
	for _, ns := range tier.Spec.Namespaces {
		refs = append(refs, tier.Status.Revisions[ns.TemplateRef])
	}
	if tier.Spec.ClusterResources != nil {
		refs = append(refs, tier.Status.Revisions[tier.Spec.ClusterResources.TemplateRef])
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
