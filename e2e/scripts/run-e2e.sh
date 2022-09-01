#!/bin/bash
set -e
set -o pipefail
set -o errexit

#TMP=$(mktemp -d)
TMP="${PWD}/e2e/tmp"

echo "Creating kind clusters..."

(kind get clusters | grep knative-edge-e2e-cloud > /dev/null) \
    && kind get kubeconfig --name knative-edge-e2e-cloud > $TMP/kubeconfig-cloud \
    || kind create cluster --name knative-edge-e2e-cloud --kubeconfig $TMP/kubeconfig-cloud --image kindest/node:v$ENVTEST_K8S_VERSION --wait 1m
(kind get clusters | grep knative-edge-e2e-edge > /dev/null) \
    && kind get kubeconfig --name knative-edge-e2e-edge > $TMP/kubeconfig-edge \
    || kind create cluster --name knative-edge-e2e-edge --kubeconfig $TMP/kubeconfig-edge --image kindest/node:v$ENVTEST_K8S_VERSION --wait 1m

echo ""
echo "Deploying metrics server..."

$KUSTOMIZE build e2e/config/metrics | KUBECONFIG=$TMP/kubeconfig-cloud kubectl apply -f -
$KUSTOMIZE build e2e/config/metrics | KUBECONFIG=$TMP/kubeconfig-edge kubectl apply -f -

echo ""
echo "Deploying Knative Operator..."

KUBECONFIG=$TMP/kubeconfig-cloud kubectl apply -f e2e/config/knative-serving/operator.yaml
KUBECONFIG=$TMP/kubeconfig-edge kubectl apply -f e2e/config/knative-serving/operator.yaml

KUBECONFIG=$TMP/kubeconfig-cloud kubectl wait deployments/knative-operator --for condition=Available=True --timeout=5m
KUBECONFIG=$TMP/kubeconfig-edge kubectl wait deployments/knative-operator --for condition=Available=True --timeout=5m

echo ""
echo "Deploying Knative Serving..."

KUBECONFIG=$TMP/kubeconfig-cloud kubectl apply -f e2e/config/knative-serving/cloud
KUBECONFIG=$TMP/kubeconfig-edge kubectl apply -f e2e/config/knative-serving/edge

KUBECONFIG=$TMP/kubeconfig-cloud kubectl wait -n knative-serving KnativeServing knative-serving --for condition=Ready=True --timeout=5m
KUBECONFIG=$TMP/kubeconfig-edge kubectl wait -n knative-serving KnativeServing knative-serving --for condition=Ready=True --timeout=5m

echo ""
echo "Deploying Knative Edge..."

$KUSTOMIZE build config/default/cloud | KO_DOCKER_REPO=kind.local KIND_CLUSTER_NAME=knative-edge-e2e-cloud ko resolve -f - | KUBECONFIG=$TMP/kubeconfig-cloud kubectl apply -f -
KUBECONFIG=$TMP/kubeconfig-cloud kubectl apply -f e2e/config/knative-edge/cloud

$KUSTOMIZE build config/default/edge | KO_DOCKER_REPO=kind.local KIND_CLUSTER_NAME=knative-edge-e2e-edge ko resolve -f - | KUBECONFIG=$TMP/kubeconfig-edge kubectl apply -f -

KUBECONFIG=$TMP/kubeconfig-edge kubectl create namespace knative-edge-e2e --dry-run=client -o yaml | KUBECONFIG=$TMP/kubeconfig-edge kubectl apply -f -

(mkdir -p $TMP/secret \
    && KUBECONFIG=$TMP/kubeconfig-cloud bash e2e/scripts/generate-kubeconfig.sh knative-edge-e2e-edge https://knative-edge-e2e-edge:6443 knative-edge-reflector knative-edge-system > $TMP/secret/kubeconfig \
    && cd $TMP/secret \
    && KUBECONFIG=$TMP/kubeconfig-edge kubectl create secret generic knative-edge-kubeconfig --from-file=./kubeconfig --dry-run=client -o yaml | KUBECONFIG=$TMP/kubeconfig-edge kubectl apply -f -)

# && KUBECONFIG=$TMP/kubeconfig-edge kubectl create secret generic -n knative-edge-e2e knative-edge-kubeconfig --from-literal=name=e2e-edge --from-file=./kubeconfig --dry-run=client -o yaml | KUBECONFIG=$TMP/kubeconfig-edge kubectl apply -f)

KUBECONFIG=$TMP/kubeconfig-edge kubectl wait deployments/knative-edge-operator --for condition=Available=True --timeout=1m

KUBECONFIG=$TMP/kubeconfig-edge kubectl apply -f e2e/config/knative-edge/edge
KUBECONFIG=$TMP/kubeconfig-edge kubectl rollout restart deployments/knative-edge-operator
KUBECONFIG=$TMP/kubeconfig-edge kubectl wait deployments/knative-edge-operator --for condition=Available=True --timeout=1m

exit 1

KUBECONFIG=$TMP/kubeconfig-edge kubectl wait -n knative-edge-system deployment/knative-edge-controller-manager --for condition=Available=True --timeout=1m

echo ""
echo "Running E2E tests..."

echo "TODO"
# rm -rf $TMP

# KUBECONFIG_CLOUD=$TMP/kubeconfig-cloud \
# KUBECONFIG_EDGE=$TMP/kubeconfig-edge \
# KUBEBUILDER_ASSETS="$shell$ENVTEST use $ENVTEST_K8S_VERSION -p path)" go test ./... -coverprofile e2e-cover.out

# echo kind delete cluster --name knative-edge-e2e-cloud
# echo kind delete cluster --name knative-edge-e2e-edge