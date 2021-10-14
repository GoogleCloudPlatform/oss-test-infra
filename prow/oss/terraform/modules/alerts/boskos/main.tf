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

# Enabled only when allowed_list is empty, this is default behavior
resource "google_monitoring_alert_policy" "boskos-alerts" {
  count        = length(var.allowed_list) == 0 ? 1 : 0
  project      = var.project
  display_name = "boskos-alerts"
  combiner     = "OR" # required

  conditions {
    display_name = "Boskos ran out of resources"

    condition_monitoring_query_language {
      duration = "0s"
      query    = <<-EOT
      fetch k8s_container
      | metric 'workload.googleapis.com/boskos_resources'
      | {
          t_0:
          filter state == 'free'
          ;
          t_1:
          ident
      }
      | group_by [metric.type]
      | outer_join 0
      | condition t_0.value_boskos_resources_aggregate == 0 && t_1.value_boskos_resources_aggregate > 5
      | window 1m
      EOT
      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "Boskos ran out of resources"
    mime_type = "text/markdown"
  }

  # gcloud beta monitoring channels list --project=oss-prow
  notification_channels = ["projects/${var.project}/notificationChannels/${var.notification_channel_id}"]
}

# Enabled only when allowed_list is not empty, not enabled by default
resource "google_monitoring_alert_policy" "boskos-alerts-selected" {
  project      = var.project
  for_each     = var.allowed_list
  display_name = "boskos-alerts-selected-${each.key}"
  combiner     = "OR" # required

  conditions {
    display_name = "Boskos ran out of resources"

    condition_monitoring_query_language {
      duration = "0s"
      query    = <<-EOT
      fetch k8s_container
      | metric 'workload.googleapis.com/boskos_resources'
      | filter metric.type = '${each.key}'
      | {
          t_0:
          filter state == 'free'
          ;
          t_1:
          ident
      }
      | outer_join 0
      | condition t_0.value.boskos_resources == 0 && t_1.value.boskos_resources > 5
      | window 1m
      EOT
      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "Boskos ran out of resources - ${each.key}"
    mime_type = "text/markdown"
  }

  # gcloud beta monitoring channels list --project=oss-prow
  notification_channels = ["projects/${var.project}/notificationChannels/${var.notification_channel_id}"]
}
