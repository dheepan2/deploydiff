package resource

import (
	"strings"
	"testing"

	"github.com/dheepan2/deploydiff/internal/manifest"
)

func TestParseNormalizesKubernetesIdentityAndMetadata(t *testing.T) {
	input := manifest.Resource{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Namespace:  "production",
		Name:       "api",
		Source:     "manifests/api.yaml",
		Document:   2,
		Object: map[string]any{
			"metadata": map[string]any{
				"labels":      map[string]any{"app": "api"},
				"annotations": map[string]any{"owner": "platform"},
			},
		},
	}

	resource, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if resource.ID != (ID{Group: "apps", Version: "v1", Kind: "Deployment", Namespace: "production", Name: "api"}) {
		t.Errorf("ID = %#v", resource.ID)
	}
	if resource.ID.String() != "apps/v1 Deployment production/api" {
		t.Errorf("ID string = %q", resource.ID.String())
	}
	if resource.Labels["app"] != "api" || resource.Annotations["owner"] != "platform" {
		t.Errorf("metadata = labels %#v annotations %#v", resource.Labels, resource.Annotations)
	}
}

func TestParseSupportsCoreAPIVersionAndEmptyMetadataMaps(t *testing.T) {
	resource, err := Parse(manifest.Resource{
		APIVersion: "v1",
		Kind:       "Namespace",
		Name:       "production",
		Object:     map[string]any{"metadata": map[string]any{}},
	})
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if resource.ID.Group != "" || resource.ID.Version != "v1" || resource.ID.String() != "v1 Namespace production" {
		t.Errorf("core ID = %#v (%q)", resource.ID, resource.ID.String())
	}
	if len(resource.Labels) != 0 || len(resource.Annotations) != 0 {
		t.Errorf("empty metadata maps = %#v, %#v", resource.Labels, resource.Annotations)
	}
}

func TestParseRejectsInvalidAPIVersionAndMetadata(t *testing.T) {
	tests := []struct {
		name  string
		input manifest.Resource
		want  string
	}{
		{name: "invalid API version", input: manifest.Resource{APIVersion: "apps/v1/extra", Kind: "Deployment", Name: "api", Object: map[string]any{"metadata": map[string]any{}}}, want: "invalid apiVersion"},
		{name: "missing metadata", input: manifest.Resource{APIVersion: "v1", Kind: "ConfigMap", Name: "config", Object: map[string]any{}}, want: "missing metadata"},
		{name: "invalid label", input: manifest.Resource{APIVersion: "v1", Kind: "ConfigMap", Name: "config", Object: map[string]any{"metadata": map[string]any{"labels": map[string]any{"version": 1}}}}, want: "must be a string"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse(test.input)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("Parse error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestParseAllRejectsDuplicateIdentity(t *testing.T) {
	inputs := []manifest.Resource{
		{APIVersion: "v1", Kind: "ConfigMap", Namespace: "production", Name: "config", Source: "one.yaml", Document: 1, Object: map[string]any{"metadata": map[string]any{}}},
		{APIVersion: "v1", Kind: "ConfigMap", Namespace: "production", Name: "config", Source: "two.yaml", Document: 2, Object: map[string]any{"metadata": map[string]any{}}},
	}

	_, err := ParseAll(inputs)
	if err == nil || !strings.Contains(err.Error(), "duplicate resource v1 ConfigMap production/config") || !strings.Contains(err.Error(), "one.yaml document 1") || !strings.Contains(err.Error(), "two.yaml document 2") {
		t.Errorf("ParseAll error = %v", err)
	}
}
