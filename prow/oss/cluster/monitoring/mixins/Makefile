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

SHELL=/bin/bash -o pipefail

include ../../../../Makefile.gcloud.mk

PROJECT       ?= oss-prow
ZONE          ?= us-west1-a
CLUSTER       ?= prow

grafana-dashboards: install
	@mkdir -p dashboards_out
	@for input in $(shell ls grafana_dashboards); do \
		output="$${input%.*}.json"; \
		echo "Generating $${output} from $${input} ..."; \
		jsonnet -J vendor/ -J lib/ "grafana_dashboards/$${input}" > "./dashboards_out/$${output}"; \
	done

prow_prometheusrule.yaml: install
	@mkdir -p prometheus_out
	@echo "Generating prow_prometheusrule.yaml ..."
	jsonnet -J lib/ ./prometheus/prow_prometheusrule.jsonnet | gojsontoyaml > prometheus_out/$@

apply-configmaps: get-cluster-credentials grafana-dashboards
	@for input in $(shell ls dashboards_out); do \
		dashboard_name="grafana-dashboard-$${input%.*}"; \
		echo "Generating dashboard $${dashboard_name} from $${input} ..."; \
		kubectl create configmap -n prow-monitoring "$${dashboard_name}" --from-file="$${input}=dashboards_out/$${input}" --dry-run -o yaml | kubectl apply -f -; \
	done

apply-prow-prometheusrule: get-cluster-credentials prow_prometheusrule.yaml
	kubectl apply -f prometheus_out/

apply: apply-configmaps apply-prow-prometheusrule

clean:
	rm -rfv ./dashboards_out/*.json
	rm -rfv ./prometheus_out/*.json
	rm -rfv ./prometheus_out/*.yaml

install:
	@if [[ -z "$$(which jsonnet)" ]]; then \
	  go get github.com/google/go-jsonnet/cmd/jsonnet; \
	fi
	@if [[ -z "$$(which gojsontoyaml)" ]]; then \
	  go get github.com/brancz/gojsontoyaml; \
	fi

.PHONY: grafana-dashboards apply-configmaps apply-prow-prometheusrule apply install clean
