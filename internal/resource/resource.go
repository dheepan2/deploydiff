// Package resource converts loaded manifests into a stable Kubernetes resource model.
package resource

import (
	"fmt"
	"strings"

	"github.com/dheepan2/deploydiff/internal/manifest"
)

// ID uniquely identifies a Kubernetes resource within a deployment state.
type ID struct {
	Group     string
	Version   string
	Kind      string
	Namespace string
	Name      string
}

// String returns a stable, human-readable identity for the resource.
func (id ID) String() string {
	apiVersion := id.Version
	if id.Group != "" {
		apiVersion = id.Group + "/" + id.Version
	}
	if id.Namespace == "" {
		return fmt.Sprintf("%s %s %s", apiVersion, id.Kind, id.Name)
	}
	return fmt.Sprintf("%s %s %s/%s", apiVersion, id.Kind, id.Namespace, id.Name)
}

// Resource is the normalized representation used by downstream analysis.
type Resource struct {
	ID          ID
	Labels      map[string]string
	Annotations map[string]string
	Source      string
	Document    int
	Object      map[string]any
}

// Parse normalizes one loaded manifest resource.
func Parse(input manifest.Resource) (Resource, error) {
	group, version, err := parseAPIVersion(input.APIVersion)
	if err != nil {
		return Resource{}, err
	}
	if input.Kind == "" {
		return Resource{}, fmt.Errorf("missing kind")
	}
	if input.Name == "" {
		return Resource{}, fmt.Errorf("missing metadata.name")
	}

	metadata, err := metadataFromObject(input.Object)
	if err != nil {
		return Resource{}, err
	}
	labels, err := stringMap(metadata, "labels")
	if err != nil {
		return Resource{}, err
	}
	annotations, err := stringMap(metadata, "annotations")
	if err != nil {
		return Resource{}, err
	}

	return Resource{
		ID: ID{
			Group:     group,
			Version:   version,
			Kind:      input.Kind,
			Namespace: input.Namespace,
			Name:      input.Name,
		},
		Labels:      labels,
		Annotations: annotations,
		Source:      input.Source,
		Document:    input.Document,
		Object:      input.Object,
	}, nil
}

// ParseAll normalizes a deployment state and rejects duplicate resource IDs.
func ParseAll(inputs []manifest.Resource) ([]Resource, error) {
	resources := make([]Resource, 0, len(inputs))
	seen := make(map[ID]Resource, len(inputs))
	for _, input := range inputs {
		resource, err := Parse(input)
		if err != nil {
			return nil, sourceError(input, err)
		}
		if previous, exists := seen[resource.ID]; exists {
			return nil, fmt.Errorf("duplicate resource %s in %s and %s", resource.ID, sourceLocation(previous.Source, previous.Document), sourceLocation(resource.Source, resource.Document))
		}
		seen[resource.ID] = resource
		resources = append(resources, resource)
	}
	return resources, nil
}

func parseAPIVersion(apiVersion string) (group, version string, err error) {
	parts := strings.Split(apiVersion, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			return "", "", fmt.Errorf("missing apiVersion")
		}
		return "", parts[0], nil
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid apiVersion %q", apiVersion)
		}
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("invalid apiVersion %q", apiVersion)
	}
}

func metadataFromObject(object map[string]any) (map[string]any, error) {
	metadata, ok := object["metadata"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}
	return metadata, nil
}

func stringMap(metadata map[string]any, field string) (map[string]string, error) {
	value, found := metadata[field]
	if !found {
		return map[string]string{}, nil
	}
	values, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("metadata.%s must be a map", field)
	}

	result := make(map[string]string, len(values))
	for key, value := range values {
		stringValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("metadata.%s[%q] must be a string", field, key)
		}
		result[key] = stringValue
	}
	return result, nil
}

func sourceError(input manifest.Resource, err error) error {
	if input.Source == "" {
		return err
	}
	return fmt.Errorf("parse resource from %s: %w", sourceLocation(input.Source, input.Document), err)
}

func sourceLocation(source string, document int) string {
	if document == 0 {
		return source
	}
	return fmt.Sprintf("%s document %d", source, document)
}
