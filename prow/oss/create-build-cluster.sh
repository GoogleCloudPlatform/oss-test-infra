#!/usr/bin/env bash
# This script is used to create a new build cluster for use with oss-prow. The cluster will have a 
# single pd-ssd nodepool that will have autoupgrade and autorepair enabled.
#
# Usage: populate the parameters by setting them below or specifying environment variables then run
# the script and follow the prompts. You'll be prompted to share some credentials and commands
# with the current oncall.


set -o errexit
set -o nounset
set -o pipefail

# Specific to Prow instance, don't change these.
export PROW_INSTANCE_NAME="${PROW_INSTANCE_NAME:-oss-prow}"
export ADMIN_IAM_MEMBER="${ADMIN_IAM_MEMBER:-group:mdb.cloud-kubernetes-engprod-oncall@google.com}"
export PROW_SECRET_ACCESSOR_SA="kubernetes-external-secrets-sa@oss-prow.iam.gserviceaccount.com"
export PROW_DEPLOYMENT_DIR="./prow/oss/cluster" # From root of repo
export CONTROL_PLANE_SA="oss-prow-public-deck@oss-prow.iam.gserviceaccount.com,oss-prow@oss-prow.iam.gserviceaccount.com"

export GITHUB_CLONE_URI="git@github.com:GoogleCloudPlatform/oss-test-infra.git"

# Specific to the build cluster
export TEAM="${TEAM:-}"
export PROJECT="${PROJECT:-${PROW_INSTANCE_NAME}-build-${TEAM}}"
export ZONE="${ZONE:-us-west1-b}"
export CLUSTER="${CLUSTER:-${PROJECT}}"
export GCS_BUCKET="${GCS_BUCKET:-gs://${PROJECT}}"

# Only needed for creating cluster
export GITHUB_FORK_URI="${GITHUB_FORK_URI:-}"  # Set this to the URI of your personal fork of this repo. Similar to GITHUB_CLONE_URI.
export MACHINE="${MACHINE:-n1-standard-8}"
export NODECOUNT="${NODECOUNT:-5}"
export DISKSIZE="${DISKSIZE:-100GB}"

# Only needed for creating project
export FOLDER_ID="${FOLDER_ID:-0123}"
export BILLING_ACCOUNT_ID="${BILLING_ACCOUNT_ID:-0123}"  # Find the billing account ID in the cloud console.

bash <(curl -sSfL https://raw.githubusercontent.com/kubernetes/test-infra/master/prow/create-build-cluster.sh) "$@"
