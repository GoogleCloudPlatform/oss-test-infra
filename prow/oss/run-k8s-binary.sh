#!/usr/bin/env bash
# This script is for running arbitrary Go binaries from the k8s/test-infra repo.
# Running this script when the binary cannot run inside a Docker container (for example,
# when the binary needs to authenticate to GCP).
# This script installs the binary directly from the k8s/test-infra repo instead of adding
# k8s.io/test-infra as a Go module inside the current directory. This is because we've seen
# dependency problems when doing so in other repos.
#
# Usage:
# $1       : The relative path for installing Go binary from k8s/test-infra
# $2..n    : The arguments for running the Go binary
#
# Example:
# - ./run-k8s-binary.sh prow/cmd/generic-autobumper --config=prow/oss/oss-autobump-config.yaml


set -o errexit
set -o nounset
set -o pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}/../..")" && pwd -P)"

TMP_DIR="$(mktemp -d)"
K8S_INFRA_DIR="${TMP_DIR}/test-infra"
BINARY="${TMP_DIR}/binary"

# Install the Go binary directly from k8s/test-infra
git clone git@github.com:kubernetes/test-infra.git "$K8S_INFRA_DIR"
cd "$K8S_INFRA_DIR"
go build -o "${BINARY}" "$1"
shift

cd $ROOT_DIR
"${BINARY}" $@
