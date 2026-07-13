package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	command := NewRootCmd(&out, &errOut)
	command.SetArgs(args)
	err := command.Execute()
	return out.String(), errOut.String(), err
}

func TestRootHelpListsCommandsAndGlobalFlags(t *testing.T) {
	out, _, err := runCommand(t, "--help")
	if err != nil {
		t.Fatalf("root help returned an error: %v", err)
	}

	for _, expected := range []string{"compare", "version", "--config", "--verbose", "--output"} {
		if !strings.Contains(out, expected) {
			t.Errorf("help output does not contain %q:\n%s", expected, out)
		}
	}
}

func TestVersionUsesBuildVersion(t *testing.T) {
	originalVersion := Version
	Version = "v0.1.0-test"
	t.Cleanup(func() { Version = originalVersion })

	out, _, err := runCommand(t, "version")
	if err != nil {
		t.Fatalf("version returned an error: %v", err)
	}
	if out != "v0.1.0-test\n" {
		t.Errorf("version output = %q, want %q", out, "v0.1.0-test\n")
	}
}

func TestVerboseLoggingAndOutputValidation(t *testing.T) {
	_, errOut, err := runCommand(t, "--verbose", "version")
	if err != nil {
		t.Fatalf("verbose version returned an error: %v", err)
	}
	if !strings.Contains(errOut, "command=deploydiff version output=table") {
		t.Errorf("verbose log = %q", errOut)
	}

	_, _, err = runCommand(t, "--output", "markdown", "version")
	if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("invalid output error = %v, want unsupported format error", err)
	}
}

func TestCompareRequiresOneSupportedInputForm(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing path", args: []string{"compare", "before", "after"}, want: "load before deployment state"},
		{name: "git references", args: []string{"compare", "--base", "origin/main", "--head", "HEAD"}, want: "Git reference comparison is not implemented yet"},
		{name: "missing head", args: []string{"compare", "--base", "origin/main"}, want: "both --base and --head are required together"},
		{name: "no inputs", args: []string{"compare"}, want: "provide two manifest paths or both --base and --head"},
		{name: "mixed inputs", args: []string{"compare", "before", "after", "--base", "origin/main", "--head", "HEAD"}, want: "provide two manifest paths or both --base and --head"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := runCommand(t, test.args...)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("compare error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestCompareLoadsManifestsAndRendersChanges(t *testing.T) {
	dir := t.TempDir()
	before := filepath.Join(dir, "before")
	after := filepath.Join(dir, "after")
	if err := os.MkdirAll(before, 0o755); err != nil {
		t.Fatalf("create before directory: %v", err)
	}
	if err := os.MkdirAll(after, 0o755); err != nil {
		t.Fatalf("create after directory: %v", err)
	}
	writeManifest(t, before, "resources.yaml", `
apiVersion: v1
kind: ConfigMap
metadata:
  name: removed
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 2
`)
	writeManifest(t, after, "resources.yaml", `
apiVersion: v1
kind: ConfigMap
metadata:
  name: added
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3
`)

	out, _, err := runCommand(t, "compare", before, after)
	if err != nil {
		t.Fatalf("compare returned an error: %v", err)
	}
	for _, expected := range []string{"Deployment Comparison", "Added (1)", "+ v1 ConfigMap added", "Removed (1)", "- v1 ConfigMap removed", "Modified (1)", "~ apps/v1 Deployment api"} {
		if !strings.Contains(out, expected) {
			t.Errorf("comparison output does not contain %q:\n%s", expected, out)
		}
	}

	jsonOut, _, err := runCommand(t, "--output", "json", "compare", before, after)
	if err != nil {
		t.Fatalf("JSON compare returned an error: %v", err)
	}
	for _, expected := range []string{`"added": [`, `"v1 ConfigMap added"`, `"removed": [`, `"modified": [`} {
		if !strings.Contains(jsonOut, expected) {
			t.Errorf("JSON output does not contain %q:\n%s", expected, jsonOut)
		}
	}
}

func writeManifest(t *testing.T, directory, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}
