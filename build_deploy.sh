#!/bin/bash

set -e

IMAGE="quay.io/cloudservices/sources-monitor-go"
IMAGE_TAG=$(git rev-parse --short=7 HEAD)

if [[ -z "$QUAY_USER" || -z "$QUAY_TOKEN" ]]; then
    echo "QUAY_USER and QUAY_TOKEN must be set"
    exit 1
fi

if [[ -z "$RH_REGISTRY_USER" || -z "$RH_REGISTRY_TOKEN" ]]; then
    echo "RH_REGISTRY_USER and RH_REGISTRY_TOKEN  must be set"
    exit 1
fi

DOCKER_CONF="$PWD/.docker"
mkdir -p "$DOCKER_CONF"

# Use --password-stdin to avoid leaking credentials to build logs (CWE-532)
echo "$QUAY_TOKEN" | docker --config="$DOCKER_CONF" login -u="$QUAY_USER" --password-stdin quay.io
echo "$RH_REGISTRY_TOKEN" | docker --config="$DOCKER_CONF" login -u="$RH_REGISTRY_USER" --password-stdin registry.redhat.io

# Enable xtrace only after credential handling is complete
set -xv
docker --config="$DOCKER_CONF" build -t "${IMAGE}:${IMAGE_TAG}" .
docker --config="$DOCKER_CONF" push "${IMAGE}:${IMAGE_TAG}"
# Backward compatibility with CI/QA
docker --config="$DOCKER_CONF" tag "${IMAGE}:${IMAGE_TAG}" "${IMAGE}:latest"
docker --config="$DOCKER_CONF" push "${IMAGE}:latest"
docker --config="$DOCKER_CONF" tag "${IMAGE}:${IMAGE_TAG}" "${IMAGE}:qa"
docker --config="$DOCKER_CONF" push "${IMAGE}:qa"
