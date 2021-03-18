# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

include ../../../Makefile.gcloud.mk

PROJECT       ?= oss-prow
ZONE          ?= us-west1-a
CLUSTER       ?= prow

apply-components: get-cluster-credentials
	kubectl apply -f ./prometheus-operator-crds
	kubectl apply -f .

apply: apply-components
	$(MAKE) -C ./mixins apply
	# Make sure keeping @ so that the token is not surfaced on command line.
	# Makefile execute each line in separate sub-shell, so all commands have to be on the same line
	@email="$(shell gcloud beta secrets versions access latest --secret prometheus-alerter-email-address --project ${PROJECT})"; \
		token="$(shell gcloud beta secrets versions access latest --secret prometheus-alerter-app-password --project ${PROJECT})"; \
		cat ./secrets/alertmanager-prow_secret.yaml | sed s/{{\ smtp_app_username\ }}/$${email}/g | \
        sed s/{{\ smtp_app_password\ }}/$${token}/g | \
		kubectl apply -f -

.PHONY: apply-components apply
