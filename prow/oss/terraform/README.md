## terraform

This directory contains terrafrom configurations for provisioning monitoring and alerting stacks on GCP for oss-prow. These configurations are applied manually fo now

### Prerequisite For Provisioning

-   Terraform 0.13.1
    [Installation guide](https://www.terraform.io/downloads.html)

-   Authenticate with GCP

    ```text
    $ gcloud auth login && gcloud auth application-default login
    ```

### Initial Setup (One time action)

This is done once before initial provisioning of monitoring and alerting stacks.

```text
    $ gsutil mb -p oss-prow gs://oss-prow-terraform
    $ gsutil versioning set on gs://oss-prow-terraform
```

### Provisioning

1.  Run `terraform init`. Terraform will automatically download the plugins
    required to execute this code. You only need to do this once per machine.

    ```text
    $ terraform init
    ```

1.  Execute Terraform:

    ```text
    $ terraform apply
    ```

### Boskos Alerts

#### Prerequisite

Existing notification channel is required for setting this up. Select by:
```
gcloud alpha monitoring channels list --project=<YOUR-PROJECT>
```

If not exist, you can go to `https://pantheon.corp.google.com/monitoring/alerting/notifications?project=<YOUR-PROJECT>` and create one.

#### Boskos Alerts for Your Project

Boskos alerts are for user projects, the easiest way to use it is by creating a `main.tf` file in your source repo:

```
module "boskos-alert" {
    source = "git::ssh://git@github.com/GoogleCloudPlatform/oss-test-infra.git//prow/oss/terraform/modules/alerts/boskos?ref=<COMMIT_SHA>"

    project = "<YOUR_PROJECT>"
    notification_channel_id = "<NOTIFICATION_CHANNEL_ID>"
}
```

Applying this will create alerts for all boskos resources managed under `<YOUR_PROJECT>`.

Alternatively, it's possible to define an allowed_list for selected boskos resources:
```
module "boskos-alert" {
    # Same as above

    allowed_list = ["resource-a", "resource-b"]
}
```
