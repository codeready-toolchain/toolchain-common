.PHONY: generate-kubefed-crd
generate-kubefed-crd: vendor
	@echo "Re-generating the KubeFed CRD..."
	$(Q)go run $(shell pwd)/vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go crd:trivialVersions=true \
	paths=./vendor/sigs.k8s.io/kubefed/pkg/apis/core/v1beta1/... output:crd:dir=deploy/crds output:stdout
	./scripts/update-kubefed-crd.sh -c deploy/crds/core.kubefed.io_kubefedclusters.yaml -s pkg/cluster/kubefedcluster_crd.go
	@rm -rf deploy/crds
