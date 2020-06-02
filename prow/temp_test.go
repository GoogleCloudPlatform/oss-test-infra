package prow_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ossDir = "oss"
)

var (
	excludedFiles = sets.NewString("bump.sh", "temp_test.go", "tests/jobs_test.go", "tests/README.md")
	excludedDir   = "prowjobs"
)

// Check to make sure that all files under `prow/oss` still exists under `prow`
func TestFileDeletion(t *testing.T) {
	if err := filepath.Walk(ossDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		oldPath, _ := filepath.Rel(ossDir, path)
		_, err = os.Lstat(oldPath)
		if err != nil {
			return fmt.Errorf("couldn't locate file %q for %q: '%v'", path, oldPath, err)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// Walking through current directory `prow`, and make sure all files underneath
// are consistent with files under `prow/oss` directory
func TestContent(t *testing.T) {
	if err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || strings.HasPrefix(path, ossDir+"/") || excludedFiles.Has(path) ||
			strings.HasPrefix(path, excludedDir+"/") {
			return nil
		}
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed reading file %q with error: '%v'", path, err)
		}
		newPath := filepath.Join(ossDir, path)
		newContent, err := ioutil.ReadFile(newPath)
		if err != nil {
			return fmt.Errorf("failed reading file %q with error: '%v'", newPath, err)
		}
		// Normalize content
		contentStr := string(content)
		newContentStr := strings.Replace(string(newContent), "prow/oss", "prow", -1)
		if diff := cmp.Diff(contentStr, newContentStr); diff != "" {
			return fmt.Errorf("diff failed. old(-), new(+): \n%s", diff)
		}
		return nil
	}); err != nil {
		t.Fatalf("Files under `prow/oss` are not consistent with files under `prow`:\n %v", err)
	}
}
