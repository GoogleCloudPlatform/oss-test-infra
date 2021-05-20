See [upstream prow](https://github.com/kubernetes/test-infra/tree/master/prow) documentation for more detailed and generic information about what prow is and how it works.

## Onboarding

If you are onboarding your GitHub organization or repository to be served by the OSS Prow instance, please see the [onboarding guide](./onboarding.md).

## Upgrading Prow

*Note*: oss prow is now automatically updated on a daily basis, see [Autobump
PRs](https://github.com/GoogleCloudPlatform/oss-test-infra/search?q=author%3Agoogle-oss-robot+is%3Apr+sort%3Aupdated+head%3Aautobump-oss-prow&type=Issues)
for ongoing/histories. This is configured as
[`ci-oss-test-infra-autobump-prow`](https://github.com/GoogleCloudPlatform/oss-test-infra/blob/49cc9a1bff81427ea8f10b9625269be7a9cf3ae0/prow/prowjobs/GoogleCloudPlatform/oss-test-infra/gcp-oss-test-infra-config.yaml#L335)
job.

Please check recent [prow announcements](https://github.com/kubernetes/test-infra/tree/master/prow#announcements) before updating, if you are not already familiar with them.

```shell
prow/bump.sh --auto
# commit change and merge PR
make -C prow/oss deploy
# kubectl get pods and watch for problems
# https://oss-prow.knative.dev and watch for problems
# Look at stack driver logs (go/oss-prow-debug or whatever) and look for problems)
```

### Watch pods

```console
$ watch kubectl get pods

Every 2.0s: kubectl get pods                                                                                                         Fri Aug 11 15:40:31 2017

NAME                         READY     STATUS    RESTARTS   AGE
deck-3621325446-00drl        1/1       Running   0          54m
deck-3621325446-9pdqw        1/1       Running   0          55m
deck-3621325446-njnwk        1/1       Running   0          54m
hook-3348033068-2tdd3        1/1       Running   0          45m
hook-3348033068-x99bf        1/1       Running   0          45m
horologium-617344823-js4mk   1/1       Running   0          50m
plank-302445171-92rfx        1/1       Running   0          41m
sinker-799599164-z44wj       1/1       Running   0          34m
tot-763621987-pktpj          1/1       Running   0          37m
```

### Check logs

```bash
kubectl logs -l app=deck # or the appropriate label like app=hook
# or a specific pod: kubectl logs deck-3621325446-00drl
```

## Creating a Job on Your Repo

### Github Trigger

The most common pattern is to trigger a job on some sort of Github event, esp. on PRs and on PR merges. Prow has concepts for these two specific stages. The first, running jobs on a PR, is called a presubmit job. The second, running jobs after the PR is merged, is called a postsubmit.

Both of these types of jobs can be configured using the config configmap [here](./config.yaml). In the configmap, you are configuring on which repo to run a particular job, basic metadata like the name, and then the build image. For these to be triggered, you must add `trigger` to the list of plugins in the plugins configmap [here](./plugins.yaml). For example, to add a simple presubmit to `my-repo`, requires the following edits:

```yaml
# in config.yaml
triggers:
- repos:
  - istio/istio
  - istio/test-infra
  - istio/<my-repo> # ADD THIS LINE
# ...
presubmits:
  # ...
  istio/<my-repo>: # ADD THIS BLOCK
  - name: my-repo-presubmit
    context: prow/my-repo-presubmit.sh
    always_run: true
    rerun_command: "@istio-testing test this"
    trigger: "@istio-testing test this"
    branches:
    - master
    spec:
      containers:
      - image: gcr.io/istio-testing/prowbazel:0.4.11
    # ...

# in plugins.yaml
my-repo: # ADD THIS BLOCK
- trigger
```

### Manually Trigger a Prow Job

```bash
# Assuming you cannot click the rerun button on deck
# and if you are oncall, do the following:
go get -u k8s.io/test-infra/prow/cmd/mkpj

mkpj --job=FOO > ~/foo.yaml # and answer interactive questions

# Contact #oncall to ensure you are approved to do the following:
kubectl --context=oss-prow create -f ~/foo.yaml
```

## Prow Secrets

Some of the prow secrets are managed by kubernetes external secrets, which
allows prow cluster creating secrets based on values from google secret manager
(Not necessarily the same GCP project where prow is located). See more detailed
instruction at [Prow Secret](https://github.com/kubernetes/test-infra/blob/master/prow/prow_secrets.md).
