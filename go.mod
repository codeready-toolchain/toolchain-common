module github.com/codeready-toolchain/toolchain-common

require (
	github.com/codeready-toolchain/api v0.0.0-20220407065959-2029a1f03cfc
	github.com/emicklei/go-restful v2.12.0+incompatible // indirect
	github.com/go-logr/logr v0.4.0
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/lestrrat-go/jwx v0.9.0
	github.com/magiconair/properties v1.8.5
	github.com/openshift/api v0.0.0-20211028023115-7224b732cc14
	// using latest commit from 'github.com/openshift/library-go@release-4.9'
	github.com/openshift/library-go v0.0.0-20220211144658-96cd7a701be1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.28.0 // indirect
	github.com/redhat-cop/operator-utils v1.3.3-0.20220121120056-862ef22b8cdf
	github.com/stretchr/testify v1.7.0
	gopkg.in/h2non/gock.v1 v1.0.14
	gopkg.in/square/go-jose.v2 v2.3.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.22.7
	k8s.io/apimachinery v0.22.7
	k8s.io/client-go v0.22.7
	sigs.k8s.io/controller-runtime v0.10.3
)

go 1.16
