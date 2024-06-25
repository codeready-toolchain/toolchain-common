package featuretoggle

import (
	"fmt"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
)

// FeatureToggleAnnotationKey generates an annotation key for the feature name.
// This key can be used in Space, NSTemplateSet, etc. CRs to indicate that the corresponding feature toggle should be enabled.
// This is the format of such keys: toolchain.dev.openshift.com/feature/<featureName>
func FeatureToggleAnnotationKey(featureName string) string {
	return fmt.Sprintf("%s%s", toolchainv1alpha1.FeatureAnnotationKeyPrefix, featureName)
}
