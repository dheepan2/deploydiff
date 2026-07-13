package diff

import (
	"strings"
	"testing"

	"github.com/dheepan2/deploydiff/internal/resource"
)

func TestCompareClassifiesAndSortsChanges(t *testing.T) {
	before := []resource.Resource{
		fixtureResource("v1", "ConfigMap", "removed", map[string]any{"value": "old"}),
		fixtureResource("apps/v1", "Deployment", "api", map[string]any{"replicas": 2}),
		fixtureResource("v1", "Service", "api", map[string]any{"port": 8080}),
	}
	after := []resource.Resource{
		fixtureResource("v1", "ConfigMap", "z-added", map[string]any{"value": "new"}),
		fixtureResource("v1", "ConfigMap", "a-added", map[string]any{"value": "new"}),
		fixtureResource("apps/v1", "Deployment", "api", map[string]any{"replicas": 3}),
		fixtureResource("v1", "Service", "api", map[string]any{"port": 8080}),
	}

	result, err := Compare(before, after)
	if err != nil {
		t.Fatalf("Compare returned an error: %v", err)
	}
	if !result.HasChanges() {
		t.Fatal("HasChanges() = false, want true")
	}
	if got := resourceNames(result.Added); strings.Join(got, ",") != "a-added,z-added" {
		t.Errorf("added = %v", got)
	}
	if got := resourceNames(result.Removed); strings.Join(got, ",") != "removed" {
		t.Errorf("removed = %v", got)
	}
	if len(result.Modified) != 1 || result.Modified[0].After.ID.Name != "api" || result.Modified[0].Before.Object["replicas"] != 2 || result.Modified[0].After.Object["replicas"] != 3 {
		t.Errorf("modified = %#v", result.Modified)
	}
}

func TestCompareIgnoresSourceMetadata(t *testing.T) {
	before := fixtureResource("v1", "Service", "api", map[string]any{"port": 8080})
	before.Source = "before/service.yaml"
	before.Document = 1
	after := fixtureResource("v1", "Service", "api", map[string]any{"port": 8080})
	after.Source = "after/service.yaml"
	after.Document = 4

	result, err := Compare([]resource.Resource{before}, []resource.Resource{after})
	if err != nil {
		t.Fatalf("Compare returned an error: %v", err)
	}
	if result.HasChanges() {
		t.Errorf("result = %#v, want no changes", result)
	}
}

func TestCompareRejectsDuplicateResources(t *testing.T) {
	duplicate := fixtureResource("v1", "ConfigMap", "config", map[string]any{"value": "one"})
	duplicate.Source = "one.yaml"
	duplicate.Document = 1
	second := duplicate
	second.Source = "two.yaml"
	second.Document = 2

	_, err := Compare([]resource.Resource{duplicate, second}, nil)
	if err == nil || !strings.Contains(err.Error(), "duplicate resource v1 ConfigMap config in before state") || !strings.Contains(err.Error(), "one.yaml document 1") || !strings.Contains(err.Error(), "two.yaml document 2") {
		t.Errorf("Compare error = %v", err)
	}
}

func fixtureResource(apiVersion, kind, name string, object map[string]any) resource.Resource {
	group, version := "", apiVersion
	if strings.Contains(apiVersion, "/") {
		parts := strings.Split(apiVersion, "/")
		group, version = parts[0], parts[1]
	}
	return resource.Resource{
		ID:     resource.ID{Group: group, Version: version, Kind: kind, Name: name},
		Object: object,
	}
}

func resourceNames(resources []resource.Resource) []string {
	names := make([]string, len(resources))
	for i, resource := range resources {
		names[i] = resource.ID.Name
	}
	return names
}
