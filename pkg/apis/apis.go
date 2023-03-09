package apis

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	routev1 "github.com/openshift/api/route/v1"
)

// AddToScheme adds all required resources to the default Scheme
func AddToScheme(s *runtime.Scheme) error {
	var AddToSchemes runtime.SchemeBuilder
	addToSchemes := append(AddToSchemes,
		toolchainv1alpha1.AddToScheme,
		corev1.AddToScheme,
		authenticationv1.AddToScheme, // used by the registration service proxy to verify cached tokens
		routev1.Install,              // used by the registration service to access proxy plugin endpoints exposed via openshift routes
	)
	return addToSchemes.AddToScheme(s)
}
