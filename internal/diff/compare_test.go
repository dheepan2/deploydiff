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
	if got := result.Modified[0].Changes; len(got) != 1 || got[0].Path != "replicas" || got[0].Before != 2 || got[0].After != 3 {
		t.Errorf("field changes = %#v", got)
	}
}

func TestCompareReportsNestedAddedRemovedAndListFields(t *testing.T) {
	before := fixtureResource("apps/v1", "Deployment", "api", map[string]any{
		"spec": map[string]any{
			"replicas":   2,
			"containers": []any{map[string]any{"name": "api", "image": "example/api:v1"}},
		},
		"metadata": map[string]any{"labels": map[string]any{"app.kubernetes.io/name": "api", "old": "remove"}},
	})
	after := fixtureResource("apps/v1", "Deployment", "api", map[string]any{
		"spec": map[string]any{
			"replicas":   3,
			"containers": []any{map[string]any{"name": "api", "image": "example/api:v2"}, map[string]any{"name": "sidecar"}},
		},
		"metadata": map[string]any{"labels": map[string]any{"app.kubernetes.io/name": "api", "new": "add"}},
	})

	result, err := Compare([]resource.Resource{before}, []resource.Resource{after})
	if err != nil {
		t.Fatalf("Compare returned an error: %v", err)
	}
	changes := result.Modified[0].Changes
	want := []string{
		"metadata.labels.new",
		"metadata.labels.old",
		"spec.containers[0].image",
		"spec.containers[1]",
		"spec.replicas",
	}
	if len(changes) != len(want) {
		t.Fatalf("field changes = %#v, want %d", changes, len(want))
	}
	for index, path := range want {
		if changes[index].Path != path {
			t.Errorf("field change %d path = %q, want %q", index, changes[index].Path, path)
		}
	}
	if changes[0].BeforePresent || !changes[0].AfterPresent || changes[1].BeforePresent != true || changes[1].AfterPresent {
		t.Errorf("added/removed field presence = %#v, %#v", changes[0], changes[1])
	}
}

func TestCompareSupportsCoreKubernetesResourceKinds(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		kind       string
		before     map[string]any
		after      map[string]any
		path       string
	}{
		{
			name:       "Ingress",
			apiVersion: "networking.k8s.io/v1",
			kind:       "Ingress",
			before:     map[string]any{"spec": map[string]any{"rules": []any{map[string]any{"host": "api.internal"}}}},
			after:      map[string]any{"spec": map[string]any{"rules": []any{map[string]any{"host": "api.example.com"}}}},
			path:       "spec.rules[0].host",
		},
		{
			name:       "Service",
			apiVersion: "v1",
			kind:       "Service",
			before:     map[string]any{"spec": map[string]any{"ports": []any{map[string]any{"port": 8080}}}},
			after:      map[string]any{"spec": map[string]any{"ports": []any{map[string]any{"port": 8443}}}},
			path:       "spec.ports[0].port",
		},
		{
			name:       "PersistentVolumeClaim",
			apiVersion: "v1",
			kind:       "PersistentVolumeClaim",
			before:     map[string]any{"spec": map[string]any{"resources": map[string]any{"requests": map[string]any{"storage": "10Gi"}}}},
			after:      map[string]any{"spec": map[string]any{"resources": map[string]any{"requests": map[string]any{"storage": "20Gi"}}}},
			path:       "spec.resources.requests.storage",
		},
		{
			name:       "ConfigMap",
			apiVersion: "v1",
			kind:       "ConfigMap",
			before:     map[string]any{"data": map[string]any{"logLevel": "info"}},
			after:      map[string]any{"data": map[string]any{"logLevel": "debug"}},
			path:       "data.logLevel",
		},
		{
			name:       "Secret",
			apiVersion: "v1",
			kind:       "Secret",
			before:     map[string]any{"data": map[string]any{"password": "old-value"}},
			after:      map[string]any{"data": map[string]any{"password": "new-value"}},
			path:       "data.password",
		},
		{
			name:       "Deployment",
			apiVersion: "apps/v1",
			kind:       "Deployment",
			before:     map[string]any{"spec": map[string]any{"replicas": 2}},
			after:      map[string]any{"spec": map[string]any{"replicas": 3}},
			path:       "spec.replicas",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := fixtureResource(test.apiVersion, test.kind, "example", test.before)
			after := fixtureResource(test.apiVersion, test.kind, "example", test.after)
			result, err := Compare([]resource.Resource{before}, []resource.Resource{after})
			if err != nil {
				t.Fatalf("Compare returned an error: %v", err)
			}
			if len(result.Modified) != 1 || len(result.Modified[0].Changes) != 1 || result.Modified[0].Changes[0].Path != test.path {
				t.Fatalf("comparison result = %#v", result)
			}
			change := result.Modified[0].Changes[0]
			if test.kind == "Secret" {
				if !change.Sensitive || change.Before != "<redacted>" || change.After != "<redacted>" {
					t.Errorf("Secret change = %#v, want redacted values", change)
				}
				return
			}
			if change.Sensitive {
				t.Errorf("non-Secret change unexpectedly marked sensitive: %#v", change)
			}
		})
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
