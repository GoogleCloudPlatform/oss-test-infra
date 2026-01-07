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
    $ gcloud storage buckets create --project prow-metrics gs://prow-metrics-terraform
    $ gcloud storage buckets update --versioning gs://prow-metrics-terraform
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

- Existing notification channel is required for setting this up. Select by:

    ```bash
    gcloud alpha monitoring channels list --project=<YOUR-PROJECT>
    ```
    If not exist, you can go to `https://pantheon.corp.google.com/monitoring/alerting/notifications?project=<YOUR-PROJECT>` and create one.

- Enable workload metrics in the cluster where Boskos is deployed:

    ```bash
    gcloud beta container clusters update <YOUR-CLUSTER-NAME> \
        --zone=<YOUR-ZONE>
        --project=<YOUR-PROJECT>
        --monitoring=SYSTEM,WORKLOAD
    ```

- Make sure that Boskos exposes port 9090:

    ```yaml
    kind: Deployment
    metadata:
        name: boskos
    spec:
        teplate:
            spec:
                containers:
                - name: boskos
                  ports:
                  - name: metrics
                    containerPort: 9090
    ```
    Add the section under `ports` in Boskos deployment file, and apply it in your cluster.

- Add `PodMonitor` for workload metrics to collect Boskos metrics, by running:

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: monitoring.gke.io/v1alpha1
    kind: PodMonitor
    metadata:
    labels:
        app: boskos
    name: boskos
    namespace: test-pods
    spec:
    podMetricsEndpoints:
    - interval: 30s
        port: metrics
        scheme: http
    namespaceSelector:
        matchNames:
        - test-pods
    selector:
        matchLabels:
        app: boskos
    EOF
    ```
    The command above assumes that Boskos is deployed within `test-pods` namespace, replace if it's not the case.

#### Boskos Alerts for Your Project

Boskos alerts are for user projects, the easiest way to set it up in your own GCP project.

Steps:

1. Creating a `main.tf` file in your source repo:

    ```terraform
    module "boskos-alert" {
        source = "git::ssh://git@github.com/GoogleCloudPlatform/oss-test-infra.git//prow/oss/terraform/modules/alerts/boskos?ref=<COMMIT_SHA>"

        project = "<YOUR_PROJECT>"
        notification_channel_id = "<NOTIFICATION_CHANNEL_ID>"
    }
    ```
1. Run `terraform init && terraform apply`

Applying this will create alerts for all boskos resources managed under `<YOUR_PROJECT>`.

Alternatively, it's possible to define an allowed_list for selected boskos resources:

```terraform
module "boskos-alert" {
    # Same as above

    allowed_list = ["resource-a", "resource-b"]
}
```
