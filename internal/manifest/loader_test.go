package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileReadsMultipleDocuments(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, "resources.yaml", `
apiVersion: v1
kind: ConfigMap
metadata:
  name: application-config
  namespace: production
---
# This empty document is ignored.
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
`)

	resources, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("loaded %d resources, want 2", len(resources))
	}

	first := resources[0]
	if first.APIVersion != "v1" || first.Kind != "ConfigMap" || first.Name != "application-config" || first.Namespace != "production" {
		t.Errorf("first resource = %#v", first)
	}
	if first.Source != path || first.Document != 1 {
		t.Errorf("first source metadata = %q document %d", first.Source, first.Document)
	}
	if resources[1].Kind != "Deployment" || resources[1].Document != 3 {
		t.Errorf("second resource = %#v, want Deployment from document 3", resources[1])
	}
}

func TestLoadDirectoryReadsYAMLFilesInPathOrder(t *testing.T) {
	dir := t.TempDir()
	first := writeFixture(t, dir, "a-config.yaml", validManifest("ConfigMap", "config"))
	writeFixture(t, dir, "ignored.txt", validManifest("Service", "ignored"))
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}
	second := writeFixture(t, nested, "b-service.yml", validManifest("Service", "api"))

	resources, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("loaded %d resources, want 2", len(resources))
	}
	if resources[0].Source != first || resources[1].Source != second {
		t.Errorf("resource sources = %q, %q; want %q, %q", resources[0].Source, resources[1].Source, first, second)
	}
}

func TestDiscoverSkipsNonKubernetesYAMLButRejectsIncompleteManifests(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "values.yaml", "image:\n  repository: example/api\n  tag: v1\n")
	writeFixture(t, dir, "workflow.yml", "name: CI\non: push\n")
	writeFixture(t, dir, "service.yaml", validManifest("Service", "api"))

	resources, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover returned an error: %v", err)
	}
	if len(resources) != 1 || resources[0].Kind != "Service" {
		t.Errorf("discovered resources = %#v", resources)
	}

	writeFixture(t, dir, "incomplete.yaml", "apiVersion: v1\nkind: ConfigMap\n")
	_, err = Discover(dir)
	if err == nil || !strings.Contains(err.Error(), "missing metadata") {
		t.Errorf("incomplete Kubernetes manifest error = %v", err)
	}
}

func TestLoadRejectsInvalidAndIncompleteDocuments(t *testing.T) {
	dir := t.TempDir()

	invalidYAML := writeFixture(t, dir, "invalid.yaml", "apiVersion: v1\n kind: ConfigMap\n")
	_, err := Load(invalidYAML)
	if err == nil || !strings.Contains(err.Error(), "decode manifest") {
		t.Errorf("invalid YAML error = %v, want decode error", err)
	}

	missingName := writeFixture(t, dir, "missing-name.yaml", "apiVersion: v1\nkind: ConfigMap\nmetadata: {}\n")
	_, err = Load(missingName)
	if err == nil || !strings.Contains(err.Error(), "missing metadata.name") {
		t.Errorf("incomplete manifest error = %v, want metadata.name error", err)
	}

	textFile := writeFixture(t, dir, "not-a-manifest.txt", "hello\n")
	_, err = Load(textFile)
	if err == nil || !strings.Contains(err.Error(), "must have a .yaml or .yml extension") {
		t.Errorf("non-YAML file error = %v", err)
	}
}

func writeFixture(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write fixture %q: %v", path, err)
	}
	return path
}

func validManifest(kind, name string) string {
	return "apiVersion: v1\nkind: " + kind + "\nmetadata:\n  name: " + name + "\n"
}
