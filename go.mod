module github.com/codeready-toolchain/toolchain-common

require (
	github.com/codeready-toolchain/api v0.0.0-20190926105251-44ed4423e3bf
	github.com/codeready-toolchain/member-operator v0.0.0-20191003074116-a0452e9a1e41
	github.com/go-logr/logr v0.1.0
	github.com/openshift/api v3.9.1-0.20190730142803-0922aa5a655b+incompatible
	github.com/operator-framework/operator-sdk v0.10.0
	github.com/pkg/errors v0.8.1
	github.com/redhat-cop/operator-utils v0.0.0-20190827162636-51e6b0c32776
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.4.0
	gopkg.in/h2non/gock.v1 v1.0.14
	k8s.io/api v0.0.0-20190925180651-d58b53da08f5
	k8s.io/apimachinery v0.0.0-20190925235427-62598f38f24e
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.2.2
	sigs.k8s.io/kubefed v0.1.0-rc2
)

// Pinned to kubernetes-1.13.1
replace (
	k8s.io/api => k8s.io/api v0.0.0-20181213150558-05914d821849
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20181213153335-0fe22c71c476
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93
	k8s.io/client-go => k8s.io/client-go v0.0.0-20181213151034-8d9ed539ba31
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.29.0
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.8.1
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20180711000925-0cf8f7e6ed1d
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.1.10
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.1.11-0.20190411181648-9d55346c2bde
)

go 1.13
