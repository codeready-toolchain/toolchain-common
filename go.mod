module github.com/codeready-toolchain/toolchain-common

require (
	github.com/codeready-toolchain/api v0.0.0-20200220052945-b24517fc619f
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/lestrrat-go/jwx v0.9.0
	github.com/magiconair/properties v1.8.0
	github.com/openshift/api v3.9.1-0.20190730142803-0922aa5a655b+incompatible
	github.com/openshift/library-go v0.0.0-20191121124438-7c776f7cc17a
	github.com/operator-framework/operator-sdk v0.11.0
	github.com/pkg/errors v0.8.1
	github.com/redhat-cop/operator-utils v0.0.0-20190827162636-51e6b0c32776
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.4.0
	gopkg.in/h2non/gock.v1 v1.0.14
	gopkg.in/square/go-jose.v2 v2.3.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.0.0
	k8s.io/apiextensions-apiserver v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.2.2
	sigs.k8s.io/kubefed v0.1.0-rc6.0.20191023070212-24d45e9f4f15
)

// Pinned to kubernetes-1.14.1
replace (
	//TODO: Update to master once merged
	github.com/codeready-toolchain/api v0.0.0-20200212082237-859fa2e44edd => github.com/tinakurian/api v0.0.0-20200219143838-d7c2cb78b67a

	k8s.io/api => k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190409021813-1ec86e4da56c
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190409023024-d644b00f3b79
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190409023720-1bc0c81fa51d
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190311093542-50b561225d70
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190409022021-00b8e31abe9d
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190510232812-a01b7d5d6c22
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.1
)

go 1.13
