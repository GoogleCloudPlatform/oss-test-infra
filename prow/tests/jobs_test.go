/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	pluginflagutil "k8s.io/test-infra/prow/flagutil/plugins"
	"k8s.io/test-infra/prow/plugins"
)

// Loaded at init time. Testing package will call flag.Parse for us.
var (
	configPath    = flag.String("config", "../oss/config.yaml", "Path to prow config")
	jobConfigPath = flag.String("job-config", "../prowjobs/", "Path to prow job config")
	pluginFlags   pluginflagutil.PluginOptions
)

func init() {
	pluginFlags.PluginConfigPathDefault = "../oss/plugins.yaml"
	pluginFlags.AddFlags(flag.CommandLine)
}

// Loaded at TestMain.
var (
	c  *config.Config
	pc *plugins.Configuration
)

func TestMain(m *testing.M) {
	cfg, err := config.Load(*configPath, *jobConfigPath, nil, "")
	if err != nil {
		fmt.Printf("Could not load config: %v\n", err)
		os.Exit(1)
	}
	c = cfg
	pluginAgent, err := pluginFlags.PluginAgent()
	if err != nil {
		fmt.Printf("Could not plugin config: %v\n", err)
		os.Exit(1)
	}
	pc = pluginAgent.Config()

	os.Exit(m.Run())
}

func TestTrustedJobs(t *testing.T) {
	const trusted = "test-infra-trusted"
	trustedPaths := sets.NewString(
		path.Join(*jobConfigPath, "GoogleCloudPlatform", "oss-test-infra", "gcp-oss-test-infra-config.yaml"),
		path.Join(*jobConfigPath, "GoogleCloudPlatform", "testgrid", "testgrid-jobs.yaml"),
	)

	// Presubmits may not use trusted clusters.
	for _, pres := range c.PresubmitsStatic {
		for _, pre := range pres {
			if pre.Cluster == trusted {
				t.Errorf("%s: presubmits cannot use trusted clusters", pre.Name)
			}
		}
	}

	// Trusted postsubmits must be defined in trustedPath
	for _, posts := range c.PostsubmitsStatic {
		for _, post := range posts {
			if post.Cluster != trusted {
				continue
			}
			if !trustedPaths.Has(post.SourcePath) {
				t.Errorf("%s defined in %s may not run in trusted cluster", post.Name, post.SourcePath)
			}
		}
	}

	// Trusted periodics must be defined in trustedPath
	for _, per := range c.AllPeriodics() {
		if per.Cluster != trusted {
			continue
		}
		if !trustedPaths.Has(per.SourcePath) {
			t.Errorf("%s defined in %s may not run in trusted cluster", per.Name, per.SourcePath)
		}
	}
}

func TestAllJobs(t *testing.T) {
	trustedPath := path.Join(*jobConfigPath, "private-inrepoconfig-configcheck")
	var repos []string
	// Presubmits may not be triggered on hidden repos.
	for repo, jobs := range c.PresubmitsStatic {
		for _, job := range jobs {
			if !strings.HasPrefix(job.SourcePath, trustedPath+"/") {
				repos = append(repos, repo)
				break
			}
		}
	}
	for repo, jobs := range c.PostsubmitsStatic {
		for _, job := range jobs {
			if !strings.HasPrefix(job.SourcePath, trustedPath+"/") {
				repos = append(repos, repo)
				break
			}
		}
	}

	for _, repo := range repos {
		for _, hiddenRepo := range c.Deck.HiddenRepos {
			if hiddenRepo == repo || strings.HasPrefix(repo, strings.TrimRight(hiddenRepo, "/")+"/") {
				t.Errorf("%q: presubmits are not allowed on hidden repos %q", repo, hiddenRepo)
			}
		}
	}
}

func TestPrivateJobs(t *testing.T) {
	const (
		private           = "private"
		checkconfigPrefix = "us-central1-docker.pkg.dev/gob-prow/prow-images/checkconfig:"
	)
	trustedPath := path.Join(*jobConfigPath, "private-inrepoconfig-configcheck")

	errorIfNotPermitted := func(t *testing.T, name, cluster string, spec *v1.PodSpec) {
		if strings.Contains(cluster, private) && (len(spec.Containers) != 1 || !strings.HasPrefix(spec.Containers[0].Image, "us-central1-docker.pkg.dev/gob-prow/prow-images/checkconfig:")) {
			t.Errorf("%s: cannot use private cluster %s", name, cluster)
		}
	}
	errIfNotTrustedPath := func(t *testing.T, cluster, p string) {
		if strings.Contains(cluster, private) && !strings.HasPrefix(p, trustedPath+"/") {
			t.Errorf("%s: cannot be outside of %s", p, trustedPath)
		}
	}

	// Presubmits may not use trusted clusters.
	for _, pres := range c.PresubmitsStatic {
		for _, pre := range pres {
			errorIfNotPermitted(t, pre.Name, pre.Cluster, pre.Spec)
			errIfNotTrustedPath(t, pre.Cluster, pre.SourcePath)
		}
	}

	// Trusted postsubmits must be defined in trustedPath
	for _, posts := range c.PostsubmitsStatic {
		for _, post := range posts {
			errorIfNotPermitted(t, post.Name, post.Cluster, post.Spec)
			errIfNotTrustedPath(t, post.Cluster, post.SourcePath)
		}
	}

	// Trusted periodics must be defined in trustedPath
	for _, per := range c.AllPeriodics() {
		errorIfNotPermitted(t, per.Name, per.Cluster, per.Spec)
		errIfNotTrustedPath(t, per.Cluster, per.SourcePath)
	}
}

// TestNoAutomaticOrExpensiveOrgWidePlugins validates that none of the
// specified orgs have plugins enabled at the org level that either:
//   1) Act automatically without prompting from the user (e.g. size plugin or
//      lgtm plugin reacting to GH reviews).
//   or
//   2) Use API rate limit to determine if action needs to be taken.
//
// We want to prevent such plugins from being enabled org wide on orgs that
// have installed the OSS Prow GitHub App where not all repos necessarily
// wish to use Prow, for example GoogleCloudPlatform.
// We don't want any automatic actions so that Prow doesn't interfere with
// repos that haven't chosed to use Prow (basically we want to silently enable)
// and we don't want to waste rate limit in such cases either.
func TestNoAutomaticOrExpensiveOrgWidePlugins(t *testing.T) {
	orgs := []string{"GoogleCloudPlatform"}
	safePlugins := sets.NewString(
		"assign",
		"cat",
		"dog",
		"golint",
		"hold",
		"label",
		"pony",
		"shrug",
		"trigger",
		"yuks",
		// Note that this is not necessarily a complete list, just the list of plugins that we have so far audited as safe.
	)

	for _, org := range orgs {
		unsafe := sets.NewString(pc.Plugins[org].Plugins...).Difference(safePlugins)
		if unsafe.Len() > 0 {
			t.Errorf("Org %q has enabled one or more plugins that have not been audited and may either act automatically or use API rate limit to determine if an event is relevant. "+
				"Either confirm the plugin is safe and update the test or enable the plugin at the repo level. Violating plugins: %q.",
				org, unsafe.List())
		}
	}
}

// TestKubeAgentsEvalLanesReadModelFromSecret holds every kube-agents eval
// lane to one way of choosing the agent's model: read from the
// kube-agents-eval-model Secret at run time, never written into the job.
// An inline model silently forks the lanes from the Secret (which is what a
// switch edits) and from each other, and has crept back in twice since the
// Secret read landed. The blocking lane's block in kube-agents-presubmits.yaml
// is the pattern every lane copies.
func TestKubeAgentsEvalLanesReadModelFromSecret(t *testing.T) {
	const (
		modelSecretName = "kube-agents-eval-model"
		modelMountPath  = "/etc/eval-model"
		pairs           = "gate|nightly|flash|alt"
		blockingLane    = "pull-kube-agents-smoke-test"
		hint            = "copy the block of pull-kube-agents-smoke-test in kube-agents-presubmits.yaml with this lane's pair (gate, nightly, flash, alt)"
	)
	kubeAgentsDir := path.Join(*jobConfigPath, "gke-labs", "kube-agents") + "/"
	var (
		// The scripts that consume MODEL_PROVIDER and MODEL_DEFAULT_NAME; a
		// job whose script runs one of them is an eval lane.
		consumer = regexp.MustCompile(`hack/ci-(deploy|eval-pr|eval-rc)\.sh`)
		// (a)+(b): every assignment of the two variables; the only one
		// allowed is the Secret read.
		assignment   = regexp.MustCompile(`^\s*(export\s+)?MODEL_(PROVIDER|DEFAULT_NAME)=`)
		providerRead = regexp.MustCompile(`^\s*MODEL_PROVIDER=.*` + modelMountPath + `/(` + pairs + `)\.provider\b`)
		nameRead     = regexp.MustCompile(`^\s*MODEL_DEFAULT_NAME=.*` + modelMountPath + `/(` + pairs + `)\.name\b`)
		override     = regexp.MustCompile(`^\s*export\s+AGENT_MODEL_OVERRIDE=(.*)$`)
		overrideFrom = regexp.MustCompile(`^"?\$\{?MODEL_DEFAULT_NAME\}?"?\s*(#.*)?$`)
		// (d) a model identifier in script text or env values. The regexp
		// also matches the name of the judge's API-key Secret, which is not a
		// model.
		modelLiteral    = regexp.MustCompile(`gemini-[0-9a-z.-]+|claude-[0-9a-z.-]+|\b(argon|opus)\b`)
		notModelLiteral = sets.NewString("gemini-secret", "gemini-api-key")
	)

	type job struct {
		name string
		spec *v1.PodSpec
	}
	var jobs []job
	for _, pres := range c.PresubmitsStatic {
		for _, pre := range pres {
			if strings.HasPrefix(pre.SourcePath, kubeAgentsDir) {
				jobs = append(jobs, job{pre.Name, pre.Spec})
			}
		}
	}
	for _, posts := range c.PostsubmitsStatic {
		for _, post := range posts {
			if strings.HasPrefix(post.SourcePath, kubeAgentsDir) {
				jobs = append(jobs, job{post.Name, post.Spec})
			}
		}
	}
	for _, per := range c.AllPeriodics() {
		if strings.HasPrefix(per.SourcePath, kubeAgentsDir) {
			jobs = append(jobs, job{per.Name, per.Spec})
		}
	}

	// A script line that is neither blank nor a comment: n is its line
	// number within its command/args element (the `- |` block in the YAML).
	type scriptLine struct {
		n    int
		text string
	}
	selected := sets.NewString()
	for _, j := range jobs {
		if j.spec == nil {
			continue
		}
		for _, container := range j.spec.Containers {
			var lines []scriptLine
			inScope := false
			for _, element := range append(append([]string{}, container.Command...), container.Args...) {
				for i, raw := range strings.Split(element, "\n") {
					text := strings.TrimSpace(raw)
					if text == "" || strings.HasPrefix(text, "#") {
						continue
					}
					lines = append(lines, scriptLine{i + 1, text})
					if consumer.MatchString(text) {
						inScope = true
					}
				}
			}
			if !inScope {
				continue
			}
			selected.Insert(j.name)
			fail := func(check, format string, args ...interface{}) {
				t.Errorf("%s: (%s) %s; %s", j.name, check, fmt.Sprintf(format, args...), hint)
			}

			firstConsumer := -1
			assigned := map[string][]int{} // variable name -> indexes into lines
			for i, line := range lines {
				if m := assignment.FindStringSubmatch(line.text); m != nil {
					assigned["MODEL_"+m[2]] = append(assigned["MODEL_"+m[2]], i)
				}
				if m := override.FindStringSubmatch(line.text); m != nil && !overrideFrom.MatchString(strings.TrimSpace(m[1])) {
					fail("a", "line %d sets AGENT_MODEL_OVERRIDE to something other than ${MODEL_DEFAULT_NAME}: %q", line.n, line.text)
				}
				if firstConsumer < 0 && consumer.MatchString(line.text) {
					firstConsumer = i
				}
				for _, literal := range modelLiteral.FindAllString(line.text, -1) {
					if !notModelLiteral.Has(literal) {
						fail("d", "line %d names a model: %q", line.n, literal)
					}
				}
			}
			for _, env := range container.Env {
				for _, literal := range modelLiteral.FindAllString(env.Value, -1) {
					if !notModelLiteral.Has(literal) {
						fail("d", "env %s names a model: %q", env.Name, literal)
					}
				}
			}

			// The one assignment of each variable must be its Secret read.
			read := map[string]int{}    // variable name -> index into lines
			pair := map[string]string{} // variable name -> the pair it reads
			for name, re, file := "MODEL_PROVIDER", providerRead, "provider"; name != ""; {
				switch idx := assigned[name]; {
				case len(idx) == 0:
					fail("b", "%s is never read from %s/<pair>.%s (assigned on: none)", name, modelMountPath, file)
				case len(idx) > 1:
					var where []string
					for _, i := range idx {
						where = append(where, fmt.Sprintf("line %d %q", lines[i].n, lines[i].text))
					}
					fail("a", "%s is assigned %d times, the Secret read must be its only assignment: %s", name, len(idx), strings.Join(where, ", "))
				default:
					if m := re.FindStringSubmatch(lines[idx[0]].text); m == nil {
						fail("a", "line %d sets %s inline instead of reading %s/<pair>.%s: %q", lines[idx[0]].n, name, modelMountPath, file, lines[idx[0]].text)
					} else {
						read[name], pair[name] = idx[0], m[1]
					}
				}
				if name == "MODEL_PROVIDER" {
					name, re, file = "MODEL_DEFAULT_NAME", nameRead, "name"
				} else {
					name = ""
				}
			}
			if len(read) == 2 {
				switch p, n := read["MODEL_PROVIDER"], read["MODEL_DEFAULT_NAME"]; {
				case pair["MODEL_PROVIDER"] != pair["MODEL_DEFAULT_NAME"]:
					fail("b", "the provider read uses pair %q and the name read pair %q: line %d %q, line %d %q", pair["MODEL_PROVIDER"], pair["MODEL_DEFAULT_NAME"], lines[p].n, lines[p].text, lines[n].n, lines[n].text)
				case p > firstConsumer || n > firstConsumer:
					fail("b", "the Secret reads (line %d %q, line %d %q) come after the first consumer, line %d %q", lines[p].n, lines[p].text, lines[n].n, lines[n].text, lines[firstConsumer].n, lines[firstConsumer].text)
				}
			}

			// (c) the Secret is mounted where the reads look.
			mounted := false
			for _, mount := range container.VolumeMounts {
				if mount.MountPath != modelMountPath {
					continue
				}
				for _, volume := range j.spec.Volumes {
					if volume.Name == mount.Name && volume.Secret != nil && volume.Secret.SecretName == modelSecretName {
						mounted = true
					}
				}
			}
			if !mounted {
				fail("c", "no volumeMount at %s backed by a volume of Secret %s", modelMountPath, modelSecretName)
			}
		}
	}
	if !selected.Has(blockingLane) {
		t.Fatalf("%s is not among the jobs under %s whose script runs an eval script (%v); the test no longer guards the blocking lane", blockingLane, kubeAgentsDir, selected.List())
	}
}
