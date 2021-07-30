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
