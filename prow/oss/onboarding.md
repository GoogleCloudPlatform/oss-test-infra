
## Set up build cluster

Create a Build cluster to host your prow jobs using [`create_build_cluster.sh`].

It isn't necessary to use a separate build cluster for every individual repo,
but each team should use their own cluster for workload/billing isolation and
management.

## Config permission for prow

Install prow GitHub app at https://github.com/apps/google-oss-prow/installations/new. Once the installation is completed, the org/repo will start sending webhook to prow, but nothing will happen until after following steps.

## (Optional) Set up Private UI

This step is required only with private repos that want a private UI. For detailed check out [Private deck instruction].

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

### (Optional) Ssh Key For Cloning

This step is only required when the target repo(s) can not be cloned anonymously.

There are two options for creating ssh key:

1. (Recommended) [deploy-keys]. Deploy keys are associated with repo instead of
   personal account, which is much easier to manage. The drawback is that one key
   only works with a single repo.
2. (For many repos) [personal ssh key]. This key can be used to clone any repo
   that the owner of the key can clone.

Follow [set up ssh key for cloning] for integrating with prow jobs.

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
[Private deck instruction]: https://github.com/kubernetes/test-infra/blob/c647ead4ae2a0d06ca8238556d2bb8cb5319120c/prow/private_deck.md
[deploy-keys]:
https://docs.github.com/en/developers/overview/managing-deploy-keys#deploy-keys
[personal ssh key]:
https://docs.github.com/en/authentication/connecting-to-github-with-ssh/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent
[set up ssh key for cloning]: https://github.com/kubernetes/test-infra/blob/b86dee86579b993b54cb295cfd77feab129d15bb/prow/pod-utilities.md#how-to-configure
[Prow config example PR]: https://github.com/GoogleCloudPlatform/oss-test-infra/pull/376
[documented Prow config]: https://github.com/kubernetes/test-infra/blob/master/prow/config/prow-config-documented.yaml
[prow job example pr]: https://github.com/GoogleCloudPlatform/oss-test-infra/pull/375
[How to add new jobs]: https://github.com/kubernetes/test-infra/tree/master/prow/jobs.md#how-to-configure-new-jobs
[Pod-utilities]: https://github.com/kubernetes/test-infra/blob/master/prow/pod-utilities.md
[Testing prow jobs]: https://github.com/kubernetes/test-infra/blob/master/prow/build_test_update.md#How-to-test-a-ProwJob
