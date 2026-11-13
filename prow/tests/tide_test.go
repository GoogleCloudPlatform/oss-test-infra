/*
Copyright 2026 Google LLC

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
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
)

func queryCoversBranch(query config.TideQuery, branch string) bool {
	return !sets.NewString(query.ExcludedBranches...).Has(branch) &&
		(len(query.IncludedBranches) == 0 || sets.NewString(query.IncludedBranches...).Has(branch))
}

func TestPipelinesDevelopmentBranchesHaveNoTideMergeRoute(t *testing.T) {
	repository := config.OrgRepo{Org: "kubeflow", Repo: "pipelines"}
	// Ignore authors and labels: no query may cover these branches, even if a
	// future PR satisfies every other condition (including a bot exception).
	for index, query := range c.Tide.Queries {
		if !query.ForRepo(repository) {
			continue
		}
		for _, branch := range []string{"master", "main"} {
			if queryCoversBranch(query, branch) {
				t.Errorf("query %d still permits Tide to merge %s:%s: %s", index, repository, branch, query.Query())
			}
		}
	}
}

func TestTidePreservesPipelinesReleaseAndOtherRepositories(t *testing.T) {
	for _, test := range []struct {
		repository config.OrgRepo
		branch     string
	}{
		{config.OrgRepo{Org: "kubeflow", Repo: "pipelines"}, "release-2.18"},
		{config.OrgRepo{Org: "kubeflow", Repo: "pipelines"}, "release-2.17"},
		{config.OrgRepo{Org: "kubeflow", Repo: "trainer"}, "master"},
		{config.OrgRepo{Org: "kubeflow", Repo: "trainer"}, "main"},
		{config.OrgRepo{Org: "GoogleCloudPlatform", Repo: "oss-test-infra"}, "master"},
	} {
		t.Run(test.repository.String()+":"+test.branch, func(t *testing.T) {
			for _, query := range c.Tide.Queries {
				if !query.ForRepo(test.repository) || !queryCoversBranch(query, test.branch) || query.Author != "" {
					continue
				}
				labels := sets.NewString(query.Labels...)
				holds := sets.NewString(query.MissingLabels...)
				if labels.HasAll("lgtm", "approved") && holds.HasAll("do-not-merge", "do-not-merge/hold") {
					return
				}
			}
			t.Fatal("no reviewed, hold-aware Tide merge route remains")
		})
	}
}
