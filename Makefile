
# Image URL to use all building/pushing image targets
BASE_CMD ?= edge.jevv.dev/cmd
PKG ?= ${BASE_CMD}/controller
IMG ?= ko://${PKG}
REPO ?= kind.local
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.24.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
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

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: env
env: ## Displays the default environment variables
	@echo "# build envs"
	@echo "BASE_CMD=${BASE_CMD}"
	@echo "PKG=${PKG}"
	@echo "IMG=${IMG}"
	@echo "REPO=${REPO}"
	@echo "# test envs"
	@echo "ENVTEST_K8S_VERSION=${ENVTEST_K8S_VERSION}"

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd paths="./pkg/apis/..." output:crd:artifacts:config=config/crd/bases

	test ! -f config/rbac/role.yaml || rm config/rbac/role.yaml

	mkdir -p config/rbac/controller/edge
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./pkg/controllers/edge/..." output:crd:artifacts:config=config/crd/bases
	test ! -f config/rbac/role.yaml || mv config/rbac/role.yaml config/rbac/controller/edge/role.yaml

	mkdir -p config/rbac/operator
	$(CONTROLLER_GEN) rbac:roleName=operator-role crd webhook paths="./pkg/controllers/operator/..." output:crd:artifacts:config=config/crd/bases
	test ! -f config/rbac/role.yaml || mv config/rbac/role.yaml config/rbac/operator/role.yaml

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

.PHONE: e2e-test
e2e-test: manifests generate fmt vet envtest kustomize ## Run e2e tests.
	@command -v kubectl > /dev/null || (echo "You must have 'kubectl' installed in order to run E2E tests."; exit 1)
	@command -v kind > /dev/null || (echo "You must have 'kind' installed in order to run E2E tests."; exit 1)
	# @$(eval TMP := $(shell mktemp -d))
	@$(eval TMP := $(shell echo "${PWD}/e2e/tmp"))

	@echo ""
	@echo "If kind fails to create the second cluster, add a '--retain' option to the command for further debugging."
	@echo "It is likely you need to increase the inotify limit."
	@echo ""

	(kind get clusters | grep knative-edge-e2e-cloud > /dev/null) \
		&& kind get kubeconfig --name knative-edge-e2e-cloud > $(TMP)/kubeconfig-cloud \
		|| kind create cluster --name knative-edge-e2e-cloud --kubeconfig $(TMP)/kubeconfig-cloud --image kindest/node:v$(ENVTEST_K8S_VERSION) --wait 1m
	(kind get clusters | grep knative-edge-e2e-edge > /dev/null) \
		&& kind get kubeconfig --name knative-edge-e2e-edge > $(TMP)/kubeconfig-edge \
		|| kind create cluster --name knative-edge-e2e-edge --kubeconfig $(TMP)/kubeconfig-edge --image kindest/node:v$(ENVTEST_K8S_VERSION) --wait 1m

	@echo ""
	@echo "Deploying metrics server..."

	$(KUSTOMIZE) build e2e/config/metrics | KUBECONFIG=$(TMP)/kubeconfig-cloud kubectl apply -f -
	$(KUSTOMIZE) build e2e/config/metrics | KUBECONFIG=$(TMP)/kubeconfig-edge kubectl apply -f -

	@echo ""
	@echo "Deploying Knative Operator..."

	KUBECONFIG=$(TMP)/kubeconfig-cloud kubectl apply -f e2e/config/knative-serving/operator.yaml
	KUBECONFIG=$(TMP)/kubeconfig-edge kubectl apply -f e2e/config/knative-serving/operator.yaml

	KUBECONFIG=$(TMP)/kubeconfig-cloud kubectl wait deployments/knative-operator --for condition=Available=True --timeout=1m
	KUBECONFIG=$(TMP)/kubeconfig-edge kubectl wait deployments/knative-operator --for condition=Available=True --timeout=1m

	@echo ""
	@echo "Deploying Knative Serving..."

	KUBECONFIG=$(TMP)/kubeconfig-cloud kubectl apply -f e2e/config/knative-serving/cloud
	KUBECONFIG=$(TMP)/kubeconfig-edge kubectl apply -f e2e/config/knative-serving/edge

	KUBECONFIG=$(TMP)/kubeconfig-cloud kubectl wait -n knative-serving KnativeServing knative-serving --for condition=Ready=True --timeout=5m
	KUBECONFIG=$(TMP)/kubeconfig-edge kubectl wait -n knative-serving KnativeServing knative-serving --for condition=Ready=True --timeout=5m

	@echo ""
	@echo "Deploying Knative Edge..."

	$(KUSTOMIZE) build config/default/cloud | KO_DOCKER_REPO=kind.local KIND_CLUSTER_NAME=knative-edge-e2e-cloud ko resolve -f - | KUBECONFIG=$(TMP)/kubeconfig-cloud kubectl apply -f -
	KUBECONFIG=$(TMP)/kubeconfig-cloud kubectl create token -n knative-edge-system knative-edge-reflector
	KUBECONFIG=$(TMP)/kubeconfig-cloud kubectl apply -f e2e/config/knative-edge/cloud

	$(KUSTOMIZE) build config/default/edge | KO_DOCKER_REPO=kind.local KIND_CLUSTER_NAME=knative-edge-e2e-edge ko resolve -f - | KUBECONFIG=$(TMP)/kubeconfig-edge kubectl apply -f -

	(mkdir -p $(TMP)/secret \
		&& KUBECONFIG=$(TMP)/kubeconfig-cloud bash e2e/scripts/generate-kubeconfig.sh knative-edge-e2e-edge https://knative-edge-e2e-edge:6443 knative-edge-reflector knative-edge-system > $(TMP)/secret/kubeconfig \
		&& cd $(TMP)/secret \
		&& KUBECONFIG=$(TMP)/kubeconfig-edge kubectl delete secret -n knative-edge-system knative-edge-edgeconfig \
		&& KUBECONFIG=$(TMP)/kubeconfig-edge kubectl create secret generic -n knative-edge-system knative-edge-edgeconfig --from-literal=name=e2e-edge --from-file=./kubeconfig)

	KUBECONFIG=$(TMP)/kubeconfig-edge kubectl apply -f e2e/config/knative-edge/edge

	exit 1

	KUBECONFIG=$(TMP)/kubeconfig-edge kubectl wait -n knative-edge-system deployment/knative-edge-controller-manager --for condition=Available=True --timeout=1m

	@echo ""
	@echo "Running E2E tests..."

	@echo "TODO"
	# rm -rf $(TMP)

	# KUBECONFIG_CLOUD=$(TMP)/kubeconfig-cloud \
	# KUBECONFIG_EDGE=$(TMP)/kubeconfig-edge \
	# KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile e2e-cover.out

	# @echo kind delete cluster --name knative-edge-e2e-cloud
	# @echo kind delete cluster --name knative-edge-e2e-edge



##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	KO_DOCKER_REPO=${REPO} ko build ${PKG}

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	KO_DOCKER_REPO=${REPO} ko run ${PKG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | KO_DOCKER_REPO=${REPO} ko delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | KO_DOCKER_REPO=${REPO} ko apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | KO_DOCKER_REPO=${REPO} ko delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: config-only
config-only: kustomize ## Display the YAML generated by kustomize
	$(KUSTOMIZE) build config/default

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.9.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -f $(KUSTOMIZE) || curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -f $(CONTROLLER_GEN) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -f $(ENVTEST) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
