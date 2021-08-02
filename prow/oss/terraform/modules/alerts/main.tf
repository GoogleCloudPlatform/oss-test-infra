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

resource "google_monitoring_alert_policy" "heartbeat-job-stale" {
  project      = var.project
  display_name = "heartbeat-job-stale"
  combiner     = "OR" # required

  conditions {
    display_name = "heartbeat-job-stale"

    condition_absent {
      filter   = "metric.type=\"workload.googleapis.com/prowjob_state_transitions\" resource.type=\"k8s_container\" metric.label.\"job_name\"=\"${var.heartbeat_job.job_name}\" metric.label.\"state\"=\"success\""
      duration = "${var.heartbeat_job.alert_interval}"

      aggregations { # required
        alignment_period     = "${var.heartbeat_job.interval}"
        per_series_aligner   = "ALIGN_RATE"
      }
    }
  }

  documentation {
    content   = " The heartbeat job `${var.heartbeat_job.job_name}` has not had a successful run in the past ${var.heartbeat_job.alert_interval} (should run every ${var.heartbeat_job.interval})."
    mime_type = "text/markdown"
  }

  # gcloud beta monitoring channels list --project=oss-prow
  notification_channels = ["projects/${var.project}/notificationChannels/${var.notification_channel_id}"]
}
