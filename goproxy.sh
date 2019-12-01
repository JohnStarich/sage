#!/usr/bin/env bash

set -ex
trap 'sleep 1; docker rm -f goproxy' EXIT
docker rm -f goproxy || echo Container goproxy does not exist
docker run \
    --name goproxy \
    --publish 9000:8081 \
    -v "$(go env GOPATH)":/go \
    goproxy/goproxy &

if [[ "$1 $2" != 'docker build' ]]; then
    echo "Not a docker build: $1 $2" >&2
    exit 2
fi

host=host.docker.internal

if [[ "$(uname)" == Linux ]]; then
    host=$(hostname -I | awk '{print $1}')
    if [[ -z "$host" ]]; then
        echo 'Failed to identify Docker host IP' >&2
        exit 1
    fi
fi

docker build \
    --build-arg GOPROXY="http://$host:9000" \
    "${@:3}"
