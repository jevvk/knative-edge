#!/bin/bash -e

CURRENT_DIR=$(pwd)
GEN_DIR=$(dirname $0)

PROJECT_MODULE="knative.dev/edge"
IMAGE_NAME="kubernetes-codegen:latest"

CUSTOM_RESOURCE_NAME="edge.knative.dev"
CUSTOM_RESOURCE_VERSION="v1"

echo "Building codegen Docker image..."
docker build -f "${GEN_DIR}/Dockerfile.codegen" \
             -t "${IMAGE_NAME}" \
             .

cmd="/go/src/k8s.io/code-generator/generate-groups.sh all \
    $PROJECT_MODULE/pkg/client \
    $PROJECT_MODULE/pkg/apis \
    $CUSTOM_RESOURCE_NAME:$CUSTOM_RESOURCE_VERSION \
    -h /go/src/k8s.io/gengo/boilerplate/no-boilerplate.go.txt"

echo "Generating client codes..."
docker run --rm \
           -e GOROOT:/go \
           -v "${PWD}:/go/src/${PROJECT_MODULE}" \
           "${IMAGE_NAME}" $cmd

echo "Running chown..."
sudo chown $USER:$USER -R ./pkg
