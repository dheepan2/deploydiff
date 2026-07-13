package gitref

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportMaterializesGitReference(t *testing.T) {
	repository := t.TempDir()
	runGit(t, repository, "init", "-q")
	runGit(t, repository, "config", "user.email", "test@example.com")
	runGit(t, repository, "config", "user.name", "DeployDiff Test")
	if err := os.MkdirAll(filepath.Join(repository, "manifests"), 0o755); err != nil {
		t.Fatalf("create manifest directory: %v", err)
	}
	contents := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: api\n"
	if err := os.WriteFile(filepath.Join(repository, "manifests", "api.yaml"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	runGit(t, repository, "add", ".")
	runGit(t, repository, "commit", "-qm", "add manifest")

	destination := filepath.Join(t.TempDir(), "snapshot")
	if err := ExportFrom(repository, "HEAD", destination); err != nil {
		t.Fatalf("export HEAD: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "manifests", "api.yaml"))
	if err != nil {
		t.Fatalf("read exported manifest: %v", err)
	}
	if string(got) != contents {
		t.Errorf("exported contents = %q, want %q", got, contents)
	}
}

func TestArchivePathRejectsTraversal(t *testing.T) {
	_, err := archivePath(t.TempDir(), "../outside.yaml")
	if err == nil || !strings.Contains(err.Error(), "unsafe archive path") {
		t.Fatalf("archivePath traversal error = %v", err)
	}
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
