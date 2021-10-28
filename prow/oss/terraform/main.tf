# Copyright 2021 Google LLC
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

# Store terraform states in GCS
terraform {
  backend "gcs" {
    bucket = "prow-metrics-terraform"
  }
}

module "dashboards" {
  source = "./modules/dashboards"

  project = "prow-metrics"
}

module "alert" {
  source = "./modules/alerts"

  project = "prow-metrics"
  heartbeat_jobs = [
    { // oss-prow
      job_name       = "ci-oss-test-infra-heartbeat"
      interval       = "300s"
      alert_interval = "1200s"
    },
    { // k8s-prow
      job_name       = "ci-test-infra-prow-checkconfig"
      interval       = "300s"
      alert_interval = "1200s"
    }
  ]
  # gcloud alpha monitoring channels list --project=prow-metrics
  # grep displayName: prow-alert-pioneer
  notification_channel_id = "1206225022273963240"
  prow_instances = {
    oss-prow = {
      crier : { namespace : "default" }
      deck : { namespace : "default" }
      ghproxy : { namespace : "default" }
      hook : { namespace : "default" }
      horologium : { namespace : "default" }
      prow-controller-manager : { namespace : "default" }
      sinker : { namespace : "default" }
      tide : { namespace : "default" }
    }
    k8s-prow = {
      crier : { namespace : "default" }
      deck : { namespace : "default" }
      ghproxy : { namespace : "default" }
      hook : { namespace : "default" }
      horologium : { namespace : "default" }
      prow-controller-manager : { namespace : "default" }
      sinker : { namespace : "default" }
      tide : { namespace : "default" }
    }
  }
  // blackbox_probers maps HTTPS hosts to the project they should be associated with.
  blackbox_probers = {
    "oss-prow.knative.dev" : "oss-prow",

    "prow.k8s.io" : "k8s-prow",
    "testgrid.k8s.io" : "k8s-prow",
    "gubernator.k8s.io" : "k8s-prow",
  }

  bot_token_hashes = [
    "5514c8081c74362c58993e5de935cb92e38cc9397e57a72883c1878cfcdd4b38" // google-oss-robot
    // Ignore k8s-ci-robot until we resolve the token remaining inaccuracies.
    // "6624f39f2213835d6c820aff41666853557f99155d23cc52cd9171bcbed3dccc" // k8s-ci-robot
  ]
  no_webhook_alert_minutes = {
    "oss-prow" = 15
    "k8s-prow" = 10
  }
}
