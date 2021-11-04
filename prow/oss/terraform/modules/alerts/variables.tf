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

variable "project" {
  type = string
}

variable "heartbeat_jobs" {
  // TODO(cjwagner): add object() specifications to type.
  type    = list(any)
  default = []
}

variable "notification_channel_id" {
  type = string
}

variable "prow_instances" {
  type = map(any)
  default = {
    "svc_not_exist" = { "namespace" : "default" }
  }
}

// blackbox_probers maps HTTPS hosts to the project they should be associated with.
variable "blackbox_probers" {
  type    = list(string)
  default = []
}

variable "bot_token_hashes" {
  type = list(string)
}

variable "no_webhook_alert_minutes" {
  type    = map(number)
  default = {}
}
