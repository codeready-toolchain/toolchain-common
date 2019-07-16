.PHONY: test
## runs the tests without coverage and excluding E2E tests
test:
	@echo "running the tests without coverage and excluding E2E tests..."
	$(Q)go test ${V_FLAG} -race $(shell go list ./... | grep -v /test/e2e) -failfast
	
.PHONY: test-ci
# runs the tests and uploads the coverage report on codecov.io
# DO NOT USE LOCALLY: must only be called by OpenShift CI when processing new PR and when a PR is merged! 
test-ci: test-with-coverage upload-codecov-report

# Output directory for coverage information
COV_DIR = $(OUT_DIR)/coverage

.PHONY: test-with-coverage
## runs the tests *with* coverage
test-with-coverage: 
	@echo "running the tests with coverage..."
	@-mkdir -p $(COV_DIR)
	@-rm $(COV_DIR)/coverage.txt
	$(Q)go test -vet off ${V_FLAG} $(shell go list ./... | grep -v /test/e2e) -coverprofile=$(COV_DIR)/coverage.txt -covermode=atomic ./...

CODECOV_TOKEN := "543cc327-510b-4e3e-9574-2c9cba1f2bc7"
COMMIT := $(shell echo $(CLONEREFS_OPTIONS) | jq '.refs[0].pulls[0].sha')
REPO_OWNER := $(shell echo $(CLONEREFS_OPTIONS) | jq '.refs[0].org')
REPO_NAME := $(shell echo $(CLONEREFS_OPTIONS) | jq '.refs[0].repo')
PULL_NUMBER := $(shell echo $(CLONEREFS_OPTIONS) | jq '.refs[0].pulls[0].number')

.PHONY: upload-codecov-report
# Uploads the test coverage reports to codecov.io. 
# DO NOT USE LOCALLY: must only be called by OpenShift CI when processing new PR and when a PR is merged! 
upload-codecov-report: 
	@echo "uploading test coverage report..."
	# Upload coverage to codecov.io. Since we don't run on a supported CI platform (Jenkins, Travis-ci, etc.), 
	# we need to provide the PR metadata explicitely using env vars used coming from https://github.com/openshift/test-infra/blob/master/prow/jobs.md#job-environment-variables
	# 
	# Also: not using the `-F unittests` flag for now as it's temporarily disabled in the codecov UI 
	# (see https://docs.codecov.io/docs/flags#section-flags-in-the-codecov-ui)
	env
	bash <(curl -s https://codecov.io/bash) \
		-t $(CODECOV_TOKEN) \
		-f $(COV_DIR)/coverage.txt \
		-C $(COMMIT) \
		-r $(REPO_OWNER)/$(REPO_NAME) \
		-P $(PULL_NUMBER) \
		-Z
	exit 1
