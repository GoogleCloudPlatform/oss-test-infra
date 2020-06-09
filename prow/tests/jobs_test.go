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
	"testing"

	"k8s.io/test-infra/prow/config"
)

var configPath = flag.String("config", "../oss/config.yaml", "Path to prow config")
var jobConfigPath = flag.String("job-config", "../prowjobs/", "Path to prow job config")

// Loaded at TestMain.
var c *config.Config

func TestMain(m *testing.M) {
	flag.Parse()

	cfg, err := config.Load(*configPath, *jobConfigPath)
	if err != nil {
		fmt.Printf("Could not load config: %v\n", err)
		os.Exit(1)
	}
	c = cfg

	os.Exit(m.Run())
}

func TestTrustedJobs(t *testing.T) {
	const trusted = "test-infra-trusted"
	trustedPath := path.Join(*jobConfigPath, "GoogleCloudPlatform", "oss-test-infra", "gcp-oss-test-infra-config.yaml")

	// Presubmits may not use trusted clusters.
	for _, pre := range c.AllPresubmits(nil) {
		if pre.Cluster == trusted {
			t.Errorf("%s: presubmits cannot use trusted clusters", pre.Name)
		}
	}

	// Trusted postsubmits must be defined in trustedPath
	for _, post := range c.AllPostsubmits(nil) {
		if post.Cluster != trusted {
			continue
		}
		if post.SourcePath != trustedPath {
			t.Errorf("%s defined in %s may not run in trusted cluster", post.Name, post.SourcePath)
		}
	}

	// Trusted periodics must be defined in trustedPath
	for _, per := range c.AllPeriodics() {
		if per.Cluster != trusted {
			continue
		}
		if per.SourcePath != trustedPath {
			t.Errorf("%s defined in %s may not run in trusted cluster", per.Name, per.SourcePath)
		}
	}
}
