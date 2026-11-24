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

var tenantClusterAllowedDirs = map[string][]string{
	"build-blueprints": {
		path.Join("GoogleCloudPlatform", "blueprints"),
	},
	// Currently unused; block all paths until an owning directory is assigned.
	"build-cloud-kubernetes-node-management-team": {},
	"build-compute-image-import": {
		path.Join("GoogleCloudPlatform", "compute-image-import"),
	},
	"build-elcarro": {
		path.Join("GoogleCloudPlatform", "elcarro-oracle-operator"),
	},
	"build-gcp-oracle-team": {
		path.Join("google", "oracle-toolkit"),
	},
	"build-gke-managed-storage": {
		path.Join("GoogleCloudPlatform", "gcs-fuse-csi-driver"),
	},
	"build-kpt-config-sync": {
		path.Join("GoogleContainerTools", "kpt-config-sync"),
	},
	"build-kube-agents": {
		path.Join("gke-labs", "kube-agents"),
	},
	"build-kubeflow": {
		"kubeflow",
	},
	"build-openshift-eng": {
		"openshift",
	},
	"espv2": {
		path.Join("GoogleCloudPlatform", "esp-v2"),
		path.Join("google", "http_pattern_matcher"),
		path.Join("google", "jwt_verify_lib"),
		path.Join("grpc-ecosystem", "grpc-httpjson-transcoding"),
	},
	"gcp-guest": {
		path.Join("GoogleCloudPlatform", "gcp-guest"),
	},
}

func validateTenantClusterIsolation(name, cluster, sourcePath, jobConfigDir string, allowedDirsByCluster map[string][]string) error {
	// "default" is the shared untrusted build cluster; "test-infra-trusted"
	// and "*private*" clusters are enforced by TestTrustedJobs and TestPrivateJobs.
	if cluster == "" || cluster == "default" || cluster == "test-infra-trusted" || strings.Contains(cluster, "private") {
		return nil
	}
	allowedDirs, ok := allowedDirsByCluster[cluster]
	if !ok {
		return fmt.Errorf("%s (%s): cluster %q is not listed in TestTenantClusterIsolation allowlist", name, sourcePath, cluster)
	}
	cleanPath := path.Clean(sourcePath)
	for _, dir := range allowedDirs {
		prefix := path.Join(jobConfigDir, dir) + "/"
		if strings.HasPrefix(cleanPath, prefix) {
			return nil
		}
	}
	return fmt.Errorf("%s defined in %s may not run in tenant cluster %q", name, sourcePath, cluster)
}

// TestTenantClusterIsolation verifies that jobs targeting a tenant build cluster
// are defined under one of the prow/prowjobs/ directories owned by that tenant.
func TestTenantClusterIsolation(t *testing.T) {
	for _, pres := range c.PresubmitsStatic {
		for _, pre := range pres {
			if err := validateTenantClusterIsolation(pre.Name, pre.Cluster, pre.SourcePath, *jobConfigPath, tenantClusterAllowedDirs); err != nil {
				t.Error(err)
			}
		}
	}
	for _, posts := range c.PostsubmitsStatic {
		for _, post := range posts {
			if err := validateTenantClusterIsolation(post.Name, post.Cluster, post.SourcePath, *jobConfigPath, tenantClusterAllowedDirs); err != nil {
				t.Error(err)
			}
		}
	}
	for _, per := range c.AllPeriodics() {
		if err := validateTenantClusterIsolation(per.Name, per.Cluster, per.SourcePath, *jobConfigPath, tenantClusterAllowedDirs); err != nil {
			t.Error(err)
		}
	}
}

func TestValidateTenantClusterIsolation(t *testing.T) {
	const baseDir = "../prowjobs/"
	cases := []struct {
		name       string
		cluster    string
		sourcePath string
		wantErr    bool
	}{
		{
			name:       "default cluster omitted",
			cluster:    "",
			sourcePath: "../prowjobs/GoogleCloudPlatform/gcp-guest/job.yaml",
		},
		{
			name:       "default cluster explicit",
			cluster:    "default",
			sourcePath: "../prowjobs/GoogleCloudPlatform/gcp-guest/job.yaml",
		},
		{
			name:       "trusted cluster delegated to TestTrustedJobs",
			cluster:    "test-infra-trusted",
			sourcePath: "../prowjobs/GoogleCloudPlatform/gcp-guest/job.yaml",
		},
		{
			name:       "private cluster delegated to TestPrivateJobs",
			cluster:    "private-cluster",
			sourcePath: "../prowjobs/GoogleCloudPlatform/gcp-guest/job.yaml",
		},
		{
			name:       "allowed tenant directory",
			cluster:    "build-elcarro",
			sourcePath: "../prowjobs/GoogleCloudPlatform/elcarro-oracle-operator/elcarro.yaml",
		},
		{
			name:       "allowed nested sub-repo directory",
			cluster:    "build-kubeflow",
			sourcePath: "../prowjobs/kubeflow/pipelines/job.yaml",
		},
		{
			name:       "allowed shared espv2 directory",
			cluster:    "espv2",
			sourcePath: "../prowjobs/google/http_pattern_matcher/job.yaml",
		},
		{
			name:       "cross-tenant cluster rejected",
			cluster:    "build-elcarro",
			sourcePath: "../prowjobs/GoogleCloudPlatform/gcp-guest/job.yaml",
			wantErr:    true,
		},
		{
			name:       "directory prefix collision rejected",
			cluster:    "gcp-guest",
			sourcePath: "../prowjobs/GoogleCloudPlatform/gcp-guest-evil/job.yaml",
			wantErr:    true,
		},
		{
			name:       "root prowjobs file rejected",
			cluster:    "build-elcarro",
			sourcePath: "../prowjobs/job.yaml",
			wantErr:    true,
		},
		{
			name:       "unused cluster rejected",
			cluster:    "build-cloud-kubernetes-node-management-team",
			sourcePath: "../prowjobs/GoogleCloudPlatform/gcp-guest/job.yaml",
			wantErr:    true,
		},
		{
			name:       "unlisted cluster rejected",
			cluster:    "build-unlisted",
			sourcePath: "../prowjobs/GoogleCloudPlatform/gcp-guest/job.yaml",
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTenantClusterIsolation(tc.name, tc.cluster, tc.sourcePath, baseDir, tenantClusterAllowedDirs)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for cluster %q in %q, got nil", tc.cluster, tc.sourcePath)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for cluster %q in %q: %v", tc.cluster, tc.sourcePath, err)
			}
		})
	}
}
