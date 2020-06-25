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

	"k8s.io/apimachinery/pkg/util/sets"
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
	var protectedClusters = map[string][]string{
		"test-infra-trusted": {
			path.Join(*jobConfigPath, "GoogleCloudPlatform", "oss-test-infra", "gcp-oss-test-infra-config.yaml"),
		},
		"knative-prow-trusted": nil, // Nothing is allowed to run on this cluster
	}

	// Presubmits may not use protected clusters.
	for _, pre := range c.AllPresubmits(nil) {
		if _, ok := protectedClusters[pre.Cluster]; ok {
			t.Errorf("%q: presubmits cannot use protected clusters %q",
				pre.Name, pre.Cluster)
		}
	}

	// Trusted postsubmits must be defined in trustedPath
	for _, post := range c.AllPostsubmits(nil) {
		ps, ok := protectedClusters[post.Cluster]
		if !ok {
			continue
		}
		if !sets.NewString(ps...).Has(post.SourcePath) {
			t.Errorf("%q defined in %q may not run in protected cluster %q",
				post.Name, post.SourcePath, post.Cluster)
		}
	}

	// Trusted periodics must be defined in trustedPath
	for _, per := range c.AllPeriodics() {
		ps, ok := protectedClusters[per.Cluster]
		if !ok {
			continue
		}
		if !sets.NewString(ps...).Has(per.SourcePath) {
			t.Errorf("%q defined in %q may not run in protected cluster %q",
				per.Name, per.SourcePath, per.Cluster)
		}
	}
}
