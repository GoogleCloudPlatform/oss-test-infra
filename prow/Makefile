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

include Makefile.gcloud.mk

# These are the usual GKE variables.
PROJECT       ?= oss-prow
PROJECT_BUILD ?= oss-prow-builds
ZONE          ?= us-west1-a
CLUSTER       ?= prow
JOB_NAMESPACE ?= test-pods

.PHONY: update-config
update-config: get-cluster-credentials
	kubectl create configmap config --from-file=config.yaml=config.yaml --dry-run -o yaml | kubectl replace configmap config -f -

.PHONY: update-plugins
update-plugins: get-cluster-credentials
	kubectl create configmap plugins --from-file=plugins.yaml=plugins.yaml --dry-run -o yaml | kubectl replace configmap plugins -f -

.PHONY: deploy
deploy: get-cluster-credentials
	kubectl apply -f ./cluster/

.PHONY: deploy-build
deploy-build: get-build-cluster-credentials
	kubectl apply -f ./cluster/build/
