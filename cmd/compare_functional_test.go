package cmd_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dheepan2/deploydiff/cmd"
)

func TestCompareFixtureDirectories(t *testing.T) {
	before := compareFixturePath(t, "before")
	after := compareFixturePath(t, "after")

	t.Run("table output", func(t *testing.T) {
		out, err := runCompare("--discover", before, after)
		if err != nil {
			t.Fatalf("compare returned an error: %v", err)
		}
		for _, expected := range []string{
			"Deployment Comparison",
			"Added (1)",
			"+ v1 ConfigMap production/application-config",
			"Removed (1)",
			"- v1 ConfigMap production/legacy-config",
			"Modified (1)",
			"~ apps/v1 Deployment production/api",
			"spec.replicas: 2 → 3",
		} {
			if !strings.Contains(out, expected) {
				t.Errorf("table output does not contain %q:\n%s", expected, out)
			}
		}
		if strings.Contains(out, "Service production/api") {
			t.Errorf("unchanged Service should not appear in output:\n%s", out)
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		out, err := runCompare("--output", "json", "--discover", before, after)
		if err != nil {
			t.Fatalf("JSON compare returned an error: %v", err)
		}

		var report struct {
			Added    []string `json:"added"`
			Removed  []string `json:"removed"`
			Modified []struct {
				Resource string `json:"resource"`
				Changes  []struct {
					Path   string `json:"path"`
					Before any    `json:"before"`
					After  any    `json:"after"`
				} `json:"changes"`
			} `json:"modified"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("decode JSON output: %v\n%s", err, out)
		}
		if strings.Join(report.Added, ",") != "v1 ConfigMap production/application-config" ||
			strings.Join(report.Removed, ",") != "v1 ConfigMap production/legacy-config" ||
			len(report.Modified) != 1 || report.Modified[0].Resource != "apps/v1 Deployment production/api" ||
			len(report.Modified[0].Changes) != 1 || report.Modified[0].Changes[0].Path != "spec.replicas" || report.Modified[0].Changes[0].Before != float64(2) || report.Modified[0].Changes[0].After != float64(3) {
			t.Errorf("comparison report = %#v", report)
		}
	})

	t.Run("no changes", func(t *testing.T) {
		out, err := runCompare("--discover", before, before)
		if err != nil {
			t.Fatalf("compare returned an error: %v", err)
		}
		if out != "No deployment changes.\n" {
			t.Errorf("no-change output = %q", out)
		}
	})
}

func runCompare(args ...string) (string, error) {
	var out, errOut bytes.Buffer
	command := cmd.NewRootCmd(&out, &errOut)
	command.SetArgs(append([]string{"compare"}, args...))
	if err := command.Execute(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func compareFixturePath(t *testing.T, state string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate functional test file")
	}
	return filepath.Join(filepath.Dir(filepath.Dir(file)), "testdata", "compare", state)
}
