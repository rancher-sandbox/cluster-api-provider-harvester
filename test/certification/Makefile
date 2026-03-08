KUBECONFIG ?= $(HOME)/.kube/config
E2E_CONFIG ?= $(shell pwd)/config/config.yaml
ARTIFACTS_FOLDER ?= $(shell pwd)/_artifacts
GINKGO_LABEL_FILTER ?= full

.PHONY: test
test: ## Run certification tests against existing cluster
	mkdir -p $(ARTIFACTS_FOLDER)
	USE_EXISTING_CLUSTER=true \
	KUBECONFIG=$(KUBECONFIG) \
	E2E_CONFIG=$(E2E_CONFIG) \
	ARTIFACTS_FOLDER=$(ARTIFACTS_FOLDER) \
	go test -v -tags e2e -timeout 60m -count 1 \
		./suites/import-gitops/ \
		-ginkgo.v \
		-ginkgo.label-filter="$(GINKGO_LABEL_FILTER)"

.PHONY: build
build: ## Compile the test binary
	go test -c -tags e2e -o bin/certification.test ./suites/import-gitops/

.PHONY: deps
deps: ## Download dependencies
	go mod tidy

.PHONY: clean
clean: ## Clean artifacts
	rm -rf _artifacts bin

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
