package hash

import (
	"encoding/json"
	"sort"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
)

// TemplateTierHashLabelKey returns the label key to specify the version of the templates of the given tier
func TemplateTierHashLabelKey(tierName string) string {
	return toolchainv1alpha1.LabelKeyPrefix + tierName + "-tier-hash"
}

// ComputeHashForNSTemplateTier computes the hash of the value of `status.revisions[]`
func ComputeHashForNSTemplateTier(tier *toolchainv1alpha1.NSTemplateTier) (string, error) {
	refs := []string{}
	for _, rev := range tier.Status.Revisions {
		refs = append(refs, rev)
	}
	return computeHash(refs)
}

type templateRefs struct {
	Refs []string `json:"refs"`
}

func computeHash(refs []string) (string, error) {
	// sort the refs to make sure we have a predictive hash!
	sort.Strings(refs)
	m, err := json.Marshal(templateRefs{Refs: refs}) // embed in a type with JSON tags
	if err != nil {
		return "", err
	}
	return Encode(m), nil
}
