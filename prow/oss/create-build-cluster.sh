#!/usr/bin/env bash
# This script is used to create a new build cluster for use with oss-prow. The cluster will have a 
# single pd-ssd nodepool that will have autoupgrade and autorepair enabled.
#
# Usage: populate the parameters by setting them below or specifying environment variables then run
# the script and follow the prompts.

set -o errexit
set -o nounset
set -o pipefail

# Specific to Prow instance, don't change these.
export PROW_INSTANCE_NAME="${PROW_INSTANCE_NAME:-oss-prow}"
export PROW_SERVICE_PROJECT="oss-prow"
export PROW_DEPLOYMENT_DIR="./prow/oss/cluster" # From root of repo
export CONTROL_PLANE_SA="oss-prow-public-deck@oss-prow.iam.gserviceaccount.com,oss-prow@oss-prow.iam.gserviceaccount.com"

# Specific to the build cluster
export TEAM="${TEAM:-}"
export PROJECT="${PROJECT:-${PROW_INSTANCE_NAME}-build-${TEAM}}"
export ZONE="${ZONE:-us-west1-b}"
export CLUSTER="${CLUSTER:-${PROJECT}}"
export GCS_BUCKET="${GCS_BUCKET:-gs://${PROJECT}}"

# Only needed for creating cluster
export MACHINE="${MACHINE:-n1-standard-8}"
export NODECOUNT="${NODECOUNT:-5}"
export DISKSIZE="${DISKSIZE:-100GB}"

# Only needed for creating project
export FOLDER_ID="${FOLDER_ID:-}"
export BILLING_ACCOUNT_ID="${BILLING_ACCOUNT_ID:-}"  # Find the billing account ID in the cloud console.
# ADMIN_IAM_MEMBER will be set as the owner of the created project, override
# this value unless it's desired for our oncall team to help debug something.
export ADMIN_IAM_MEMBER="${ADMIN_IAM_MEMBER:-group:mdb.cloud-kubernetes-engprod-oncall@google.com}"

# The following is based on / sourced from https://github.com/kubernetes-sigs/prow/blob/main/pkg/create-build-cluster.sh

# Require bash version >= 4.4
if ((${BASH_VERSINFO[0]}<4)) || ( ((${BASH_VERSINFO[0]}==4)) && ((${BASH_VERSINFO[1]}<4)) ); then
  echo "ERROR: This script requires a minimum bash version of 4.4, but got version of ${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}"
  if [ "$(uname)" = 'Darwin' ]; then
    echo "On macOS with homebrew 'brew install bash' is sufficient."
  fi
  exit 1
fi

# Macos specific settings
SED="sed"
if command -v gsed &>/dev/null; then
  SED="gsed"
fi
if ! (${SED} --version 2>&1 | grep -q GNU); then
  # darwin is great (not)
  echo "!!! GNU sed is required.  If on OS X, use 'brew install gnu-sed'." >&2
  return 1
fi

# Create temp dir to work in and clone k/t-i

origdir="$( pwd -P )"
tempdir="$( mktemp -d )"
echo
echo "Temporary files produced are stored at: ${tempdir}"
echo
cd "${tempdir}"
git clone https://github.com/kubernetes/test-infra --depth=1
cd "${origdir}"

ROOT_DIR="${tempdir}/test-infra"

function main() {
  parseArgs "$@"
  ensureProject
  ensureBucket
  ensureCluster
  ensureUploadSA
  genConfig
  gencreds
  echo "All done!"
}
# Prep and check args.
function parseArgs() {
  for var in TEAM PROJECT ZONE CLUSTER MACHINE NODECOUNT DISKSIZE; do
    if [[ -z "${!var}" ]]; then
      echo "Must specify ${var} environment variable (or specify a default in the script)."
      exit 2
    fi
    echo "${var}=${!var}"
  done
  if [[ "${PROW_INSTANCE_NAME}" != "k8s-prow" ]]; then
    if [[ "${PROW_DEPLOYMENT_DIR}" == "./config/prow/cluster" ]]; then
      read -r -n1 -p "${PROW_DEPLOYMENT_DIR} is k8s-prow specific, are you sure this is the same for ${PROW_INSTANCE_NAME} ? [y/n] "
      if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 2
      fi
    fi
  fi
}
function prompt() {
  local msg="$1" cmd="$2"
  echo
  read -r -n1 -p "$msg ? [y/n] "
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    "$cmd"
  else
    echo "Skipping and continuing to next step..."
  fi
}
function pause() {
  read -n 1 -s -r
}

authed=""
function getClusterCreds() {
  if [[ -z "${authed}" ]]; then
    gcloud container clusters get-credentials --project="${PROJECT}" --zone="${ZONE}" "${CLUSTER}"
    authed="true"
  fi
}
function ensureProject() {
  if gcloud projects describe ${PROJECT}; then
    echo "GCP project '${PROJECT}' exists, skip creating."
    return
  fi

  prompt "Failed to describe the project ${PROJECT}, press Y/y to create the project" echo
  # Create project, configure billing, enable GKE, add IAM rule for oncall team.
  echo "Creating project '${PROJECT}' (this may take a few minutes)..."
  gcloud projects create "${PROJECT}" --name="${PROJECT}" --folder="${FOLDER_ID}"
  gcloud beta billing projects link "${PROJECT}" --billing-account="${BILLING_ACCOUNT_ID}"
  gcloud services enable "container.googleapis.com" --project="${PROJECT}"
  gcloud projects add-iam-policy-binding "${PROJECT}" --member="${ADMIN_IAM_MEMBER}" --role="roles/owner"
}
function ensureCluster() {
  if gcloud container clusters describe "${CLUSTER}" --project="${PROJECT}" --zone="${ZONE}" >/dev/null 2>&1; then
    echo "Cluster '${CLUSTER}' exists in zone '${ZONE}' in project '${PROJECT}', skip creating."
    return
  else
    prompt "Pressing Y/y to create the cluster" echo
    echo "Creating cluster '${CLUSTER}' (this may take a few minutes)..."
    echo "If this fails due to insufficient project quota, request more at https://console.cloud.google.com/iam-admin/quotas?project=${PROJECT}"
    echo
    gcloud container clusters create "${CLUSTER}" --project="${PROJECT}" --zone="${ZONE}" --machine-type="${MACHINE}" --num-nodes="${NODECOUNT}" --disk-size="${DISKSIZE}" --disk-type="pd-ssd" --enable-autoupgrade --enable-autorepair --workload-pool="${PROJECT}.svc.id.goog"
  fi

  getClusterCreds
  kubectl create namespace "test-pods" --dry-run=client -o yaml | kubectl apply -f -
}

function createBucket() {
  gcloud storage buckets create "${GCS_BUCKET}" --project="${PROJECT}" --uniform-bucket-level-access
  for i in ${CONTROL_PLANE_SA//,/ }
  do
    gcloud storage buckets add-iam-policy-binding "${GCS_BUCKET}" --member="serviceAccount:${i}" --role="roles/storage.objectAdmin"
  done
}

function ensureBucket() {
  if ! gcloud storage ls "${GCS_BUCKET}"; then
    createBucket
  else
    echo "Bucket '${GCS_BUCKET}' already exists, skip creation."
  fi
}

function ensureUploadSA() {
  getClusterCreds
  local sa="prowjob-default-sa"
  local saFull="${sa}@${PROJECT}.iam.gserviceaccount.com"
  # Create a GCP service account for uploading to GCS
  if ! gcloud beta iam service-accounts describe "${saFull}" --project="${PROJECT}" >/dev/null 2>&1; then
    gcloud beta iam service-accounts create "${sa}" --project="${PROJECT}" --description="Default SA for ProwJobs to use to upload job results to GCS." --display-name="ProwJob default SA"
  else
    echo "Service account '${sa}' already exists, skip creation."
  fi
  # Ensure workload identity is enabled on the cluster
  if ! gcloud container clusters describe ${CLUSTER} --project=${PROJECT} --zone=${ZONE} | grep "${CLUSTER}.svc.id.goog" >/dev/null 2>&1; then
    "${ROOT_DIR}/workload-identity/enable-workload-identity.sh" "${PROJECT}" "${ZONE}" "${CLUSTER}"
  else
    echo "Workload identity is enabled on cluster '${CLUSTER}', skip enabling."
  fi

  # Create a k8s service account to associate with the GCP service account
  if ! kubectl -n test-pods get serviceaccount ${sa}; then
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    iam.gke.io/gcp-service-account: ${saFull}
  name: ${sa}
  namespace: test-pods
EOF
  fi

  echo "Binding GCP service account with k8s service account via workload identity. Propagation and validation may take a few minutes..."
  if ! gcloud iam service-accounts get-iam-policy --project=gob-prow prowjob-default-sa@gob-prow.iam.gserviceaccount.com | grep "${CLUSTER}.svc.id.goog[test-pods/${saFull}]" >/dev/null 2>&1; then
    "${ROOT_DIR}/workload-identity/bind-service-accounts.sh" "${PROJECT}" "${ZONE}" "${CLUSTER}" test-pods "${sa}" "${saFull}"
  fi

  # Try to authorize SA to upload to GCS_BUCKET. If this fails, the bucket if
  # probably a shared result bucket and oncall will need to handle.
  if ! gcloud storage buckets get-iam-policy "${GCS_BUCKET}" | grep "serviceAccount:${saFull}" >/dev/null 2>&1; then
    if ! gcloud storage buckets add-iam-policy-binding "${GCS_BUCKET}" --member="serviceAccount:${saFull}" --role="roles/storage.objectAdmin"; then
      echo
      echo "It doesn't look you have permission to authorize access to this bucket. This is expected for the default job result bucket."
      echo "If this is a default job result bucket, please ask the test-infra oncall (https://go.k8s.io/oncall) to run the following:"
      echo "  gcloud storage buckets add-iam-policy-binding \"${GCS_BUCKET}\" --member=\"serviceAccount:${saFull}\" --role=\"roles/storage.objectAdmin\""
      echo
      echo "Press any key to acknowledge (this doesn't need to be completed to continue this script, but it needs to be done before uploading will work)..."
      pause
    fi
  fi
}

function genConfig() {
  # TODO: Automatically inject this into config.yaml at the same time as kubeconfig credential setup (which auto creates a PR we can include this in).
  echo
  echo "The following changes should be made to the Prow instance's config.yaml file (Probably located at ${PROW_DEPLOYMENT_DIR}/../config.yaml)."
  echo
  echo "Append the following entry to the end of the slice at field 'plank.default_decoration_config_entries': "
  cat <<EOF
  - cluster: $(cluster_alias)
    config:
      gcs_configuration:
        bucket: "${GCS_BUCKET#"gs://"}"
      default_service_account_name: "prowjob-default-sa" # Use workload identity
      gcs_credentials_secret: ""                         # rather than service account key secret
EOF
  echo
  echo "Press any key to acknowledge... This doesn't need to be merged to continue this script, but it needs to be done before configuring jobs for the cluster."
  pause
}

function gencreds() {
  # Grant the Prow control plane access to the cluster.
  for i in ${CONTROL_PLANE_SA//,/ }
  do
    gcloud projects add-iam-policy-binding "${PROJECT}" --member="serviceAccount:${i}" --role="roles/container.developer"
  done

  # Generate entries to add to the kubeconfigs.yaml file.
  local kubeconfigs="$(git rev-parse --show-toplevel)/${PROW_DEPLOYMENT_DIR}/kubeconfigs/kubeconfigs.yaml"
  export KUBECONFIG="${tempdir}/kubeconfig.yaml" 
  authed="" # Force getClusterCreds to run again with the temporary KUBECONFIG.
  getClusterCreds

  echo
  echo "The following changes should be made to the Prow instance's kubeconfigs.yaml file (located at ${kubeconfigs})."
  echo "Append the following entry to the 'clusters' section: "
  echo
  grep -B 1 -A 2 certificate-authority-data "${KUBECONFIG}"
  echo
  echo "Append the following entry to the 'contexts' section: "
  echo
  cat <<EOF
  - context:
      cluster: $(kubectl config current-context)
      user: gke-auth-plugin
    name: $(cluster_alias)
EOF
  echo
  echo "ProwJobs that intend to use this cluster should specify 'cluster: $(cluster_alias)'" # TODO: color this
  echo
  echo "Press any key to acknowledge (this doesn't need to be completed to continue this script, but it needs to be done before Prow can schedule jobs to your cluster)..."
  pause
}

cluster_alias() {
  echo "build-${TEAM}"
}
gsm_secret_name() {
  echo "prow_build_cluster_kubeconfig_$(cluster_alias)"
}

function cleanup() {
  returnCode="$?"
  rm -f "sa-key.json" || true
  rm -rf "${tempdir}" || true
  exit "${returnCode}"
}
trap cleanup EXIT
main "$@"
cleanup