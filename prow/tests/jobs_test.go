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
		checkconfigPrefix = "gcr.io/k8s-prow/checkconfig:"
	)
	trustedPath := path.Join(*jobConfigPath, "private-inrepoconfig-configcheck")

	errorIfNotPermitted := func(t *testing.T, name, cluster string, spec *v1.PodSpec) {
		if strings.Contains(cluster, private) && (len(spec.Containers) != 1 || !strings.HasPrefix(spec.Containers[0].Image, "gcr.io/k8s-prow/checkconfig:")) {
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

// Knative cluster is not meant to run any prow job from this repo
func TestKnativeCluster(t *testing.T) {
	const protected = "knative-prow-trusted"
	verifyFunc := func(t *testing.T, jobName, cluster string) {
		if cluster == protected {
			t.Errorf("%s: cannot use knative cluster", jobName)
		}
	}

	for _, pres := range c.PresubmitsStatic {
		for _, pre := range pres {
			verifyFunc(t, pre.Name, pre.Cluster)
		}
	}

	for _, posts := range c.PostsubmitsStatic {
		for _, post := range posts {
			verifyFunc(t, post.Name, post.Cluster)
		}
	}

	for _, per := range c.AllPeriodics() {
		verifyFunc(t, per.Name, per.Cluster)
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
