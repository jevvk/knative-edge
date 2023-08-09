#!bash

set -e

ROOTDIR=$(dirname "$(readlink -fm "$0")")

(

cd $ROOTDIR

experiment=$1

if [[ "$experiment" == "" ]]; then
    experiment=$(openssl rand -hex 12)
fi

sed -i "s/const Version = \".*\"/const Version = \"$experiment\"/" pkg/worker/version.go
sed -i "s|image: europe-west4-docker.pkg.dev/jappy-8418e/jappy/\(.*\):.*|image: europe-west4-docker.pkg.dev/jappy-8418e/jappy/\\1:$experiment|" config/services.yaml

KO_OPTIONS="-t $experiment,latest" bash build.sh

echo $experiment

)
