#!bash

set -e

ROOTDIR=$(dirname "$(readlink -fm "$0")")

(

cd $ROOTDIR

export KO_DOCKER_REPO=europe-west4-docker.pkg.dev/jappy-8418e/jappy
# KO_OPTIONS="-L $KO_OPTIONS"
KO_OPTIONS="$KO_OPTIONS"


set -x

ko build $KO_OPTIONS -B \
    --platform linux/amd64,linux/arm64,linux/arm \
    function/cmd/helloworld-go \
    function/cmd/worker-image-processing \
    function/cmd/worker-matrix-multiply \
    function/cmd/worker-random-io 2>/dev/null

{ set +x; } 2>/dev/null

)