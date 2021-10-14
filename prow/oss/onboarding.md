
## Set up build cluster

Create a Build cluster to host your prow jobs using [`create_build_cluster.sh`].

It isn't necessary to use a separate build cluster for every individual repo,
but each team should use their own cluster for workload/billing isolation and
management.

## Config permission for prow

Install prow GitHub app at https://github.com/apps/google-oss-prow/installations/new. Once the installation is completed, the org/repo will start sending webhook to prow, but nothing will happen until after following steps.

## Config prow to act on webhooks from github

Create a pullrequest, contains:

-   Setup tide to help automatically merging PRs.
-   Enable the `trigger` plugin to allow presubmit and postsubmit ProwJobs to be triggered.
-   Enable other useful plugins.

We have catalogs of the available [plugins](https://oss-prow.knative.dev/plugins) and [commands](https://oss-prow.knative.dev/command-help) to browse through.
If not sure what setting or which plugins to use, feel free to just start with the
same set of settings for `google/exposure-notification-server` in
[Prow config example PR]. 
The [documented Prow config] is a useful reference for config fields.

After this step, prow is capable of reacting with `/` style command such as:

-   `/lgtm`: add `lgtm` label on PR.
-   `/hold`: add `do-not-merge/hold` label on PR.
-   `/retest`: rerun presubmit jobs that failed.

Tide will also start automatically merging PRs in your org/repo when
conditions are met, in the `google/exposure-notification-server` example above
these conditions are:

-   CLA signed.
-   PR contains `lgtm` and `approve` label.

To make prow/tide require presubmit tests to pass, proceed with `Adding Prow
jobs` below.

## Adding Prow jobs

Create a pullrequest, contains:

-   Add a directory named `<your_org>/<your_repo>` under `prow/prowjobs` in
    [oss prow repo].
-   Add an `OWNERS` file under `<your_org>` or `<your_org>/<your_repo>`
    directory, which contains members from your team.
-   Add prowjobs under `<your_org>/<your_repo>` in uniquely named `.yaml` files.
    File names must be unique across the Prow instance so it is advised to include
    the repo in the file name. e.g. `gcp-oss-test-infra-presubmits.yaml`

Check the [prow job example pr] for ideas.

Tip: If you want to know more about how to configure your job, please reference
to [How to add new jobs] and [Pod-utilities].

## Testing jobs

There are a few ways that you can test changes to your ProwJobs, check out
[Testing prow jobs] for more information.

## Viewing Test Results

To view all test results please navigate to https://oss-prow.knative.dev. Presubmit and postsubmit
jobs will also report to GitHub by default.

[Test Infra oncall]: https://go.k8s.io/oncall
[oss prow repo]: https://github.com/GoogleCloudPlatform/oss-test-infra
[`google-oss-robot`]: https://github.com/google-oss-robot
[`create_build_cluster.sh`]: https://github.com/GoogleCloudPlatform/oss-test-infra/blob/master/prow/oss/create-build-cluster.sh
[webhook example pr]: https://github.com/GoogleCloudPlatform/oss-test-infra/pull/547
[post-oss-test-infra-reconcile-hmacs]: https://oss-prow.knative.dev/?job=post-oss-test-infra-reconcile-hmacs
[Prow config example PR]: https://github.com/GoogleCloudPlatform/oss-test-infra/pull/376
[documented Prow config]: https://github.com/kubernetes/test-infra/blob/master/prow/config/prow-config-documented.yaml
[prow job example pr]: https://github.com/GoogleCloudPlatform/oss-test-infra/pull/375
[How to add new jobs]: https://github.com/kubernetes/test-infra/tree/master/prow/jobs.md#how-to-configure-new-jobs
[Pod-utilities]: https://github.com/kubernetes/test-infra/blob/master/prow/pod-utilities.md
[Testing prow jobs]: https://github.com/kubernetes/test-infra/blob/master/prow/build_test_update.md#How-to-test-a-ProwJob
