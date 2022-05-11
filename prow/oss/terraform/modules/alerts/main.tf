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

locals {
  sinker_monitoring_resources = {
    pods : "sinker_pods_removed"
    prowjobs : "sinker_prow_jobs_cleaned"
  }
  // Flatten var.prow_instances into a map.
  // https://www.terraform.io/docs/language/functions/flatten.html#flattening-nested-structures-for-for_each
  project_indexed_components = { for elem in
    flatten([
      for project, components in var.prow_instances : [
        for component, details in components : {
          project   = project,
          component = component,
          namespace = details.namespace
        }
      ]
    ]) :
    "${elem.project}/${elem.component}" => elem
  }
}

resource "google_monitoring_alert_policy" "sinker-alerts" {
  project      = var.project
  for_each     = local.sinker_monitoring_resources
  display_name = "sinker-not-deleting-${each.key}"
  combiner     = "OR" # required

  conditions {
    display_name = "Sinker not deleting ${each.key}"

    condition_monitoring_query_language {
      duration = "300s"
      query    = <<-EOT
      fetch k8s_container
      | metric 'workload.googleapis.com/${each.value}'
      | group_by [resource.project_id], 1h, [value_sinker_removed_sum: sum(value.${each.value})]
      | every 1h
      | condition val() < 1
      EOT
      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "Sinker not deleting any ${each.key} in an hour."
    mime_type = "text/markdown"
  }

  # gcloud beta monitoring channels list --project=oss-prow
  notification_channels = ["projects/${var.project}/notificationChannels/${var.notification_channel_id}"]
}

resource "google_monitoring_alert_policy" "predicted-gh-rate-limit-exhaustion" {
  project      = var.project
  display_name = "predicted-gh-rate-limit-exhaustion"
  combiner     = "OR" # required

  conditions {
    display_name = "predicted-gh-rate-limit-exhaustion"

    condition_monitoring_query_language {
      # This query calculates the expected remaining rate limit tokens at the
      # end of the current rate limit reset window based on rate of consumption
      # over the last 10 minutes. The alert fires if we predict that we will
      # come within 250 tokens of the limit (which is 5% of the 5000 limit).
      duration = "60s"
      query    = <<-EOT
      {t_0: # The remaining tokens in the rate limit reset window
          fetch k8s_container::workload.googleapis.com/github_token_usage
          ;
      t_1: # The expected tokens that will be used over the remaining time in the rate limit reset window based on recent usage.
          fetch k8s_container
          | {
              metric 'workload.googleapis.com/github_token_usage'
              | value 5000 - val()
              | align rate(1m) # Align rate over 1m and filter before actual 15m aggregation to drop counter resets in gauge.
              | filter val() > 0
              | align mean_aligner(15m)
              ;
              metric 'workload.googleapis.com/github_token_reset'
              | value val() / (1000000000)
          }
          | outer_join 0
          | mul
      }
      | outer_join 0
      | sub # Result is the expected remaining tokens at the end of the rate limit reset window.
      | every 1m
      | filter metric.token_hash =~ "${join("|", var.bot_token_hashes)}"
      | condition val() < 250
      | window 1m
      EOT
      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "One of the GitHub tokens used with `ghproxy` is predicted to exhaust its rate limit before the end of the rate limit reset window."
    mime_type = "text/markdown"
  }

  # gcloud beta monitoring channels list --project=oss-prow
  notification_channels = ["projects/${var.project}/notificationChannels/${var.notification_channel_id}"]
}

resource "google_monitoring_alert_policy" "pod-crashlooping" {
  for_each     = local.project_indexed_components
  project      = var.project
  display_name = "pod-crashlooping-${each.key}"
  combiner     = "OR" # required

  conditions {
    display_name = "pod-crashlooping-${each.key}"

    condition_monitoring_query_language {
      # Alert if the service crashlooped, which results in restarts with exponential backoff.
      # This threshold is higher than the default crashloop backoff threshold, mainly due to
      # the fact prow components would crashloop when kubernetes master is upgrading, which
      # normally takes 5 minutes. Setting this to 12 minutes for excluding this case
      duration = "720s"
      query    = <<-EOT
      fetch k8s_container
      | metric 'kubernetes.io/container/restart_count'
      | filter
          (resource.project_id == '${each.value.project}' && resource.container_name == '${each.value.component}' && resource.namespace_name == '${each.value.namespace}')
      | align delta(6m)
      | every 6m
      | group_by [], [value_restart_count_aggregate: aggregate(value.restart_count)]
      | condition val() > 0 '1'
      EOT
      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "The service `${each.key}` has been restarting for more than 6 minutes, very likely crashlooping."
    mime_type = "text/markdown"
  }

  # gcloud beta monitoring channels list --project=oss-prow
  notification_channels = ["projects/${var.project}/notificationChannels/${var.notification_channel_id}"]
}


resource "google_monitoring_alert_policy" "heartbeat-job-stale" {
  for_each     = { for job in var.heartbeat_jobs : job.job_name => job }
  project      = var.project
  display_name = "heartbeat-job-stale/${each.key}"
  combiner     = "OR" # required

  conditions {
    display_name = "heartbeat-job-stale/${each.key}"

    condition_monitoring_query_language {
      duration = each.value.alert_interval
      query    = <<-EOT
      fetch k8s_container
      | metric 'workload.googleapis.com/prowjob_state_transitions'
      | filter
          (metric.job_name == '${each.value.job_name}'
          && metric.state == 'success')
      | sum # Combining values reported by all prow-controller-manager pods
      | align delta_gauge(${each.value.interval})
      | every ${each.value.interval}
      | condition val() == 0
      EOT
      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = " The heartbeat job `${each.value.job_name}` has not had a successful run in the past ${each.value.alert_interval} (should run every ${each.value.interval})."
    mime_type = "text/markdown"
  }

  # gcloud beta monitoring channels list --project=oss-prow
  notification_channels = ["projects/${var.project}/notificationChannels/${var.notification_channel_id}"]
}

resource "google_monitoring_alert_policy" "probers" {
  project = var.project

  display_name = "HostDown"
  combiner     = "OR"
  conditions {
    display_name = "Host is unreachable"
    condition_monitoring_query_language {
      duration = "120s"
      query    = <<-EOT
      fetch uptime_url
      | metric 'monitoring.googleapis.com/uptime_check/check_passed'
      | align next_older(1m)
      | filter resource.project_id == '${var.project}'
      | every 1m
      | group_by [resource.host],
          [value_check_passed_not_count_true: count_true(not(value.check_passed))]
      | condition val() > 1 '1'
      EOT
      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "Host Down"
    mime_type = "text/markdown"
  }

  # gcloud beta monitoring channels list --project=oss-prow
  notification_channels = ["projects/${var.project}/notificationChannels/${var.notification_channel_id}"]
}


resource "google_monitoring_alert_policy" "webhook-missing" {
  for_each     = var.no_webhook_alert_minutes
  project      = var.project
  display_name = "webhook-missing/${each.key}"
  combiner     = "OR" # required

  conditions {
    display_name = "webhook-missing/${each.key}"

    condition_monitoring_query_language {
      duration = "${each.value * 60}s"
      query    = <<-EOT
      fetch k8s_container
      | metric 'workload.googleapis.com/prow_webhook_counter'
      | filter resource.project_id == '${each.key}'
      | sum
      | align delta_gauge(1m)
      | every 1m
      | value add [hour: end().timestamp_to_string("%H", "America/Los_Angeles").string_to_int64]
      | value add [day_of_week: end().timestamp_to_string("%u", "America/Los_Angeles").string_to_int64]
      | value add [is_weekend: if(day_of_week >= 6, 1, 0)]
      | value add [is_business_hour: if((hour >= 9) && (hour < 17), 1, 0)]
      | condition val(0) == 0 && is_business_hour * (1-is_weekend) == 1
      EOT
      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "${each.key} has received no webhooks for ${each.value} minutes during work hours."
    mime_type = "text/markdown"
  }

  # gcloud beta monitoring channels list --project=oss-prow
  notification_channels = ["projects/${var.project}/notificationChannels/${var.notification_channel_id}"]
}

resource "google_monitoring_alert_policy" "KES-Secret-Sync-Error" {
  project      = var.project
  display_name = "Kubernetes External Secret: Secret-Sync-Error"
  combiner     = "OR" # required

  conditions {
    display_name = "Secret-Sync-Error"

    condition_monitoring_query_language {
      duration = "0s"
      query    = <<-EOT
      fetch k8s_container
      | metric 'workload.googleapis.com/kubernetes_external_secrets_sync_calls_count'
      | align delta(60s)
      | filter metric.status != "success"
      | condition val() > 1.5
      EOT
      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "Kubernetes External Secrets has encountered errors while syncing."
    mime_type = "text/markdown"
  }

  # gcloud beta monitoring channels list --project=oss-prow
  notification_channels = ["projects/${var.project}/notificationChannels/${var.notification_channel_id}"]
}
