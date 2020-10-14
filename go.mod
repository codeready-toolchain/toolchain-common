module github.com/codeready-toolchain/toolchain-common

require (
	github.com/codeready-toolchain/api v0.0.0-20200914140452-d2c86086863e
	github.com/davecgh/go-spew v1.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/emicklei/go-restful v2.12.0+incompatible // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.7 // indirect
	github.com/go-openapi/swag v0.19.9 // indirect
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/google/go-cmp v0.5.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lestrrat-go/jwx v0.9.0
	github.com/magiconair/properties v1.8.1
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/ginkgo v1.13.0 // indirect
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/openshift/library-go v0.0.0-20191121124438-7c776f7cc17a
	github.com/operator-framework/operator-sdk v0.19.2
	github.com/pkg/errors v0.9.1
	github.com/redhat-cop/operator-utils v0.0.0-20190827162636-51e6b0c32776
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/tools v0.0.0-20200812195022-5ae4c3c160a0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/h2non/gock.v1 v1.0.14
	gopkg.in/square/go-jose.v2 v2.3.0
	gotest.tools v2.2.0+incompatible
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20200427153329-656914f816f9 // indirect
	k8s.io/utils v0.0.0-20200414100711-2df71ebbae66 // indirect
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200821140346-b94c46af3f2b // Using 'github.com/openshift/api@release-4.5'
	k8s.io/client-go => k8s.io/client-go v0.18.3 // Required by prometheus-operator
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6 // avoids case-insensitive import collision: "github.com/googleapis/gnostic/openapiv2" and "github.com/googleapis/gnostic/OpenAPIv2"
)

go 1.14
