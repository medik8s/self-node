# SHELL defines bash so all the inline scripts here will work as expected.
SHELL := /bin/bash

# versions at  https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_GEN_VERSION = v0.10.0

# GO_VERSION refers to the version of Golang to be downloaded when running dockerized version
GO_VERSION = 1.19

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23

# versions at https://github.com/operator-framework/operator-sdk/releases
OPERATOR_SDK_VERSION = v1.25.3

# versions at https://github.com/operator-framework/operator-registry/releases
OPM_VERSION = v1.26.2

# versions at https://github.com/kubernetes-sigs/kustomize/releases
KUSTOMIZE_VERSION = v4.5.7

OPERATOR_NAME ?= self-node-remediation

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
DEFAULT_VERSION := 0.0.1
VERSION ?= $(DEFAULT_VERSION)
export VERSION

CHANNELS = stable
export CHANNELS
DEFAULT_CHANNEL = stable
export DEFAULT_CHANNEL

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle. 
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_REGISTRY used to indicate the registery/group for the operator, bundle and catalog
IMAGE_REGISTRY ?= quay.io/medik8s
export IMAGE_REGISTRY

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# medik8s/self-node-remediation-bundle:$(IMAGE_TAG) and medik8s/self-node-remediation-catalog:$(IMAGE_TAG).
IMAGE_TAG_BASE ?= $(IMAGE_REGISTRY)/$(OPERATOR_NAME)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-operator-catalog:$(IMAGE_TAG)

# When no version is set, use latest as image tags
ifeq ($(VERSION), $(DEFAULT_VERSION))
IMAGE_TAG = latest
else
IMAGE_TAG = v$(VERSION)
endif
export IMAGE_TAG

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-operator-bundle:$(IMAGE_TAG)

# Image URL to use all building/pushing image targets
export IMG ?= $(IMAGE_TAG_BASE)-operator:$(IMAGE_TAG)

# Get the currently used golang install path (in GOPATH/bin.old, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Use kubectl, fallback to oc
KUBECTL = kubectl
ifeq (,$(shell which kubectl))
KUBECTL=oc
endif

# Run go in a container
# --rm                                                          = remove container when stopped
# -v $$(pwd):/home/go/src/github.com/medik8s/self-node-remediation-operator = bind mount current dir in container
# -u $$(id -u)                                                  = use current user (else new / modified files will be owned by root)
# -w /home/go/src/github.com/medik8s/self-node-remediation-operator         = working dir
# -e ...                                                        = some env vars, especially set cache to a user writable dir
# --entrypoint /bin bash ... -c                                 = run bash -c on start; that means the actual command(s) need be wrapped in double quotes, see e.g. check target which will run: bash -c "make test"
export DOCKER_GO=docker run --rm -v $$(pwd):/home/go/src/github.com/medik8s/$(OPERATOR_NAME)-operator \
	-u $$(id -u) -w /home/go/src/github.com/medik8s/$(OPERATOR_NAME)-operator \
	-e "GOPATH=/go" -e "GOFLAGS=-mod=vendor" -e "XDG_CACHE_HOME=/tmp/.cache" \
	-e "VERSION=$(VERSION)" -e "IMAGE_REGISTRY=$(IMAGE_REGISTRY)" \
	--entrypoint /bin/bash golang:$(GO_VERSION) -c

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen protoc ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations. Also generate protoc / gRPC code
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	PATH=$(PATH):$(shell pwd)/bin/proto/bin && $(PROTOC) --go_out=. --go-grpc_out=. pkg/peerhealth/peerhealth.proto

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

verify-no-changes: ## verify there are no un-staged changes
	./hack/verify-diff.sh

# CI uses a non-writable home dir, make sure .cache is writable
ifeq ("${HOME}", "/")
HOME=/tmp
endif

BIN_ASSETS_DIR=$(shell pwd)/bin
ENVTEST_ASSETS_DIR = ${BIN_ASSETS_DIR}/setup-envtest
ENVTEST = $(shell pwd)/bin/setup-envtest

test: envtest manifests generate fmt vet ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path --bin-dir $(PROJECT_DIR)/testbin)" \
		KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT="60s"\
		go test ./api/... ./controllers/... ./pkg/... -coverprofile cover.out -v

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: check ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | envsubst | $(KUBECTL) apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete -f -


CONTROLLER_GEN_BIN_FOLDER = $(shell pwd)/bin/controller-gen
CONTROLLER_GEN = $(CONTROLLER_GEN_BIN_FOLDER)/$(CONTROLLER_GEN_VERSION)/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
ifeq (,$(wildcard $(CONTROLLER_GEN)))
	@{ \
	rm -rf $(CONTROLLER_GEN_BIN_FOLDER) ;\
	mkdir -p $(dir $(CONTROLLER_GEN)) ;\
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION}) ;\
	}
endif

KUSTOMIZE_BIN_FOLDER = $(shell pwd)/bin/kustomize
KUSTOMIZE = $(KUSTOMIZE_BIN_FOLDER)/$(KUSTOMIZE_VERSION)/kustomize
kustomize: ## Download kustomize locally if necessary.
ifeq (,$(wildcard $(KUSTOMIZE)))
	@{ \
	rm -rf $(KUSTOMIZE_BIN_FOLDER) ;\
	mkdir -p $(dir $(KUSTOMIZE)) ;\
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@${KUSTOMIZE_VERSION}) ;\
	}
endif

.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-install-tool,$(ENVTEST_ASSETS_DIR),sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20220407132358-188b48630db2) # no tagged versions :/

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
BIN_DIR=$$(dirname $(1)) ;\
mkdir -p $$BIN_DIR ;\
echo "Downloading $(2)" ;\
GOBIN=$$BIN_DIR GOFLAGS='' go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

DEFAULT_ICON_BASE64 := $(shell base64 --wrap=0 ./config/assets/snr_icon_blue.png)
export ICON_BASE64 ?= ${DEFAULT_ICON_BASE64}
.PHONY: bundle
bundle: manifests operator-sdk kustomize ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | envsubst | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-k8s
bundle-k8s: bundle ## Generate bundle manifests and metadata customized to k8s community release, then validate generated files.
	# Note that k8s 1.25+ needs PSA label
	sed -r -i "s|by default\.|by default.\n    Note that prior to installing SNR on a Kubernetes 1.25+ cluster, a user must manually set a [privileged PSA label](https://kubernetes.io/docs/tasks/configure-pod-container/enforce-standards-namespace-labels/) on SNR's namespace. It gives SNR's agents permissions to reboot the node (in case it needs to be remediated).|;" ./bundle/manifests/$(OPERATOR_NAME).clusterserviceversion.yaml


.PHONY: bundle-update
bundle-update:
    # update container image in the metadata
	sed -r -i "s|containerImage: .*|containerImage: ${IMG}|;" ./bundle/manifests/$(OPERATOR_NAME).clusterserviceversion.yaml
	# set creation date
	sed -r -i "s|createdAt: \".*\"|createdAt: \"`date "+%Y-%m-%d %T" `\"|;" ./bundle/manifests/$(OPERATOR_NAME).clusterserviceversion.yaml
	# set skipRange
	sed -r -i "s|olm.skipRange: .*|olm.skipRange: '>=0.4.0 <${VERSION}'|;" ./bundle/manifests/$(OPERATOR_NAME).clusterserviceversion.yaml
	# set icon (not version or build date related, but just to not having this huge data permanently in the CSV)
	sed -r -i "s|base64data:.*|base64data: ${ICON_BASE64}|;" ./bundle/manifests/$(OPERATOR_NAME).clusterserviceversion.yaml
	$(OPERATOR_SDK) bundle validate ./bundle


.PHONY: bundle-build
bundle-build: bundle bundle-update ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	docker push $(BUNDLE_IMG)

.PHONY: protoc ## Download protoc (protocol buffers tool needed for gRPC)
PROTOC = $(shell pwd)/bin/proto/bin/protoc
protoc: protoc-gen-go protoc-gen-go-grpc
	test -f ${PROTOC} || (cd $(shell pwd)/bin/proto && curl -sSLo protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v3.16.0/protoc-3.16.0-linux-x86_64.zip && unzip protoc.zip && rm protoc.zip)

PROTOC_GEN_GO = $(shell pwd)/bin/proto/bin/protoc-gen-go
protoc-gen-go: ## Download protoc-gen-go locally if necessary.
	$(call go-install-tool,$(PROTOC_GEN_GO),google.golang.org/protobuf/cmd/protoc-gen-go@v1.26)

PROTOC_GEN_GO_GRPC = $(shell pwd)/bin/proto/bin/protoc-gen-go-prpc
protoc-gen-go-grpc: ## Download protoc-gen-go-grpc locally if necessary.
	$(call go-install-tool,$(PROTOC_GEN_GO_GRPC),google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0)

.PHONY: e2e-test
e2e-test:
	# KUBECONFIG must be set to the cluster, and PP needs to be deployed already
    # count arg makes the test ignoring cached test results
	go test ./e2e -ginkgo.v -ginkgo.progress -test.v -timeout 60m -count=1

.PHONY: operator-sdk
OPERATOR_SDK_BIN_FOLDER = ./bin/operator-sdk
OPERATOR_SDK = $(OPERATOR_SDK_BIN_FOLDER)/$(OPERATOR_SDK_VERSION)/operator-sdk
operator-sdk: ## Download operator-sdk locally if necessary.
ifeq (,$(wildcard $(OPERATOR_SDK)))
	@{ \
	set -e ;\
	rm -rf $(OPERATOR_SDK_BIN_FOLDER) ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	OS=linux && ARCH=amd64 && \
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$${OS}_$${ARCH} ;\
	chmod +x $(OPERATOR_SDK) ;\
	}
endif

.PHONY: opm
OPM_BIN_FOLDER = ./bin/opm
OPM = $(OPM_BIN_FOLDER)/$(OPM_VERSION)/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
	@{ \
	set -e ;\
	rm -rf $(OPM_BIN_FOLDER) ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=linux && ARCH=amd64 && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/$(OPM_VERSION)/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	mkdir ${OPM_BIN_FOLDER}/latest ;\
	cp ${OPM} ${OPM_BIN_FOLDER}/latest ;\
	}
endif

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMG to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif
# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMG) $(FROM_INDEX_OPT)

.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

##@ Targets used by CI
.PHONY: check
# WORKAROUND: add "protoc" as dependency for downloading binary first, the golang image misses the needed "unzip"tool
check: protoc ## Dockerized version of make test
	$(DOCKER_GO) "make test"

.PHONY: container-build
container-build: ## Build containers
	make docker-build bundle bundle-build

.PHONY: container-push
container-push: ## Push containers (NOTE: catalog can't be build before bundle was pushed)
	make docker-push bundle-push catalog-build catalog-push

.PHONY:vendor
vendor: ## Runs go mod vendor
	go mod vendor

.PHONY: tidy
tidy: ## Runs go mod tidy
	go mod tidy


.PHONY:verify-vendor
verify-vendor:tidy vendor verify-no-changes ##Verifies vendor and tidy didn't cause changes


.PHONY:verify-bundle
verify-bundle: manifests bundle verify-no-changes ##Verifies bundle and manifests didn't cause changes