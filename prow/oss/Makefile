# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

include ../Makefile.gcloud.mk

# These are the usual GKE variables.
PROJECT       ?= oss-prow
PROJECT_BUILD ?= oss-prow-builds
ZONE          ?= us-west1-a
CLUSTER       ?= prow
CLUSTER_BUILD ?= prow
JOB_NAMESPACE ?= test-pods

.PHONY: update-config
update-config: get-cluster-credentials
	kubectl create configmap config --from-file=config.yaml=config.yaml --dry-run -o yaml | kubectl replace configmap config -f -

.PHONY: update-plugins
update-plugins: get-cluster-credentials
	kubectl create configmap plugins --from-file=plugins.yaml=plugins.yaml --dry-run -o yaml | kubectl replace configmap plugins -f -

.PHONY: update-kubeconfigs
update-kubeconfigs:
	kubectl create configmap kubeconfigs --from-file=kubeconfigs.yaml=cluster/kubeconfigs/kubeconfigs.yaml --dry-run=client -o yaml | kubectl apply -f -

.PHONY: deploy
deploy: get-cluster-credentials update-kubeconfigs
# Apply the ProwJob CRD with --server-side=true due to its size.
	kubectl apply --server-side=true -f ./cluster/prowjob-crd/
	kubectl apply -f ./cluster/
	kubectl apply -f ./serviceaccounts/

.PHONY: deploy-build
deploy-build: get-build-cluster-credentials
	kubectl apply -f ./cluster/build/

.PHONY: deploy-monitoring
deploy-monitoring:
	$(MAKE) -C ./cluster/monitoring apply

# The targets below are for breaking glass usages, should not run on a regular
# basis

# Run fix-gencred-refresher will generate kubeconfig for "trusted" build cluster
# and store it in GSM, you will need "container.admin" and "secretManager.admin"
# permission on k8s-prow GCP project to perform this action.
# This target is for solving the bootstrapping problem of gencred refresher,
# where it runs in "trusted" build cluster, as a prow job to rotate the
# kubeconfig for "trusted" build cluster along with other build clusters. So
# when the refresher job failed for more than 2 days, prow will stop working
# with "trusted" build cluster and thus not able to schedule gencred refrsher
# prow jobs any more. And this is when the admin of this repo need to run this
# target manually.
fix-gencred-refresher:
	./run-k8s-binary.sh ./gencred --config=./prow/oss/gencred-config/gencred-config.yaml --gke-filter=projects/oss-prow/locations/us-west1-a/clusters/prow

.PHONY: fix-gencred-refresher
