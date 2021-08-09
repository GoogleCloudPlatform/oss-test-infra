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
        bucket = "oss-prow-terraform"
    }
}

module "dashboards" {
    source = "./modules/dashboards"

    project = "oss-prow"
}

module "alert" {
    source = "./modules/alerts"

    project = "oss-prow"
    heartbeat_job = {
        job_name = "ci-oss-test-infra-heartbeat"
        interval = "300s"
        alert_interval = "1200s"
    }
    notification_channel_id = "14031735832803168422"
    prow_components = {
        "deck" = {"namespace": "default"}
        "hook" = {"namespace": "default"}
        "prow-controller-manager" = {"namespace": "default"}
        "sinker" = {"namespace": "default"}
        "tide" = {"namespace": "default"}
    }
}
