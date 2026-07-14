// Package manifest loads Kubernetes resources from YAML files.
package manifest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Resource is a Kubernetes resource read from a manifest document.
type Resource struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
	Source     string
	Document   int
	Object     map[string]any
}

// Load reads Kubernetes resources from one YAML file or every YAML file in a
// directory tree. Files are processed in lexical path order for deterministic
// results, and multi-document YAML files are supported.
func Load(path string) ([]Resource, error) {
	return load(path, false)
}

// Discover reads Kubernetes resources from a YAML file or directory tree while
// ignoring valid YAML documents that are not Kubernetes manifests. A document
// is considered Kubernetes-looking when it has kind or metadata; incomplete
// Kubernetes-looking documents still return a validation error. apiVersion
// alone is not enough because Helm Chart.yaml files use that field too. For
// unrendered templates, only static Kubernetes identity fields are discovered.
func Discover(path string) ([]Resource, error) {
	return load(path, true)
}

func load(path string, discover bool) ([]Resource, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect manifest path %q: %w", path, err)
	}

	if !info.IsDir() {
		if !isYAML(path) {
			return nil, fmt.Errorf("manifest file %q must have a .yaml or .yml extension", path)
		}
		return loadFile(path, discover)
	}

	files, err := yamlFiles(path)
	if err != nil {
		return nil, err
	}

	var resources []Resource
	for _, file := range files {
		loaded, err := loadFile(file, discover)
		if err != nil {
			return nil, err
		}
		resources = append(resources, loaded...)
	}
	return resources, nil
}

func yamlFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && isYAML(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read manifest directory %q: %w", dir, err)
	}
	sort.Strings(files)
	return files, nil
}

func loadFile(path string, discover bool) ([]Resource, error) {
	var reader io.Reader
	if discover {
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read manifest file %q: %w", path, err)
		}
		if hasTemplateActions(contents) {
			return discoverTemplateIdentities(path, contents), nil
		}
		reader = bytes.NewReader(contents)
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open manifest file %q: %w", path, err)
		}
		defer file.Close()
		reader = file
	}

	decoder := yaml.NewDecoder(reader)
	var resources []Resource
	for document := 1; ; document++ {
		var object map[string]any
		err := decoder.Decode(&object)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode manifest %q document %d: %w", path, document, err)
		}
		if len(object) == 0 {
			continue
		}

		resource, err := resourceFromObject(object)
		if err != nil {
			if discover && !looksKubernetes(object) {
				continue
			}
			return nil, fmt.Errorf("validate manifest %q document %d: %w", path, document, err)
		}
		resource.Source = path
		resource.Document = document
		resources = append(resources, resource)
	}
	return resources, nil
}

func hasTemplateActions(contents []byte) bool {
	return bytes.Contains(contents, []byte("{{")) && bytes.Contains(contents, []byte("}}"))
}

type templateIdentity struct {
	apiVersion     string
	kind           string
	name           string
	namespace      string
	apiVersionSeen bool
	kindSeen       bool
	nameSeen       bool
	namespaceSeen  bool
	metadata       bool
	metadataIndent int
	metadataChild  int
	valid          bool
}

func discoverTemplateIdentities(path string, contents []byte) []Resource {
	document := 1
	identity := newTemplateIdentity()
	var resources []Resource

	flush := func() {
		if resource, ok := identity.resource(path, document); ok {
			resources = append(resources, resource)
		}
		identity = newTemplateIdentity()
	}

	for _, line := range strings.Split(string(contents), "\n") {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent == 0 && isDocumentBoundary(trimmed) {
			flush()
			document++
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || isTemplateDirective(trimmed) {
			continue
		}

		if indent == 0 {
			identity.metadata = false
			identity.metadataChild = -1
			switch {
			case strings.HasPrefix(trimmed, "apiVersion:"):
				identity.capture(trimmed, "apiVersion", &identity.apiVersion, &identity.apiVersionSeen)
			case strings.HasPrefix(trimmed, "kind:"):
				identity.capture(trimmed, "kind", &identity.kind, &identity.kindSeen)
			case trimmed == "metadata:" || strings.HasPrefix(trimmed, "metadata: #"):
				identity.metadata = true
				identity.metadataIndent = indent
			}
			continue
		}

		if !identity.metadata || indent <= identity.metadataIndent {
			continue
		}
		if identity.metadataChild == -1 {
			identity.metadataChild = indent
		}
		if indent != identity.metadataChild {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "name:"):
			identity.capture(trimmed, "name", &identity.name, &identity.nameSeen)
		case strings.HasPrefix(trimmed, "namespace:"):
			identity.capture(trimmed, "namespace", &identity.namespace, &identity.namespaceSeen)
		}
	}
	flush()
	return resources
}

func newTemplateIdentity() templateIdentity {
	return templateIdentity{metadataChild: -1, valid: true}
}

func (identity *templateIdentity) capture(line, key string, destination *string, seen *bool) {
	if *seen {
		identity.valid = false
		return
	}
	*seen = true
	value, ok := staticScalar(line, key)
	if !ok {
		identity.valid = false
		return
	}
	*destination = value
}

func (identity templateIdentity) resource(path string, document int) (Resource, bool) {
	if !identity.valid || !identity.apiVersionSeen || !identity.kindSeen || !identity.nameSeen {
		return Resource{}, false
	}
	metadata := map[string]any{"name": identity.name}
	if identity.namespaceSeen {
		metadata["namespace"] = identity.namespace
	}
	object := map[string]any{
		"apiVersion": identity.apiVersion,
		"kind":       identity.kind,
		"metadata":   metadata,
	}
	resource, err := resourceFromObject(object)
	if err != nil {
		return Resource{}, false
	}
	resource.Source = path
	resource.Document = document
	return resource, true
}

func staticScalar(line, key string) (string, bool) {
	raw := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), key+":"))
	if raw == "" || strings.Contains(raw, "{{") || strings.Contains(raw, "}}") {
		return "", false
	}
	var value any
	if err := yaml.Unmarshal([]byte(raw), &value); err != nil {
		return "", false
	}
	stringValue, ok := value.(string)
	return stringValue, ok && stringValue != ""
}

func isTemplateDirective(line string) bool {
	return strings.HasPrefix(line, "{{") && strings.HasSuffix(line, "}}")
}

func isDocumentBoundary(line string) bool {
	return line == "---" || strings.HasPrefix(line, "--- #")
}

func looksKubernetes(object map[string]any) bool {
	_, hasKind := object["kind"]
	_, hasMetadata := object["metadata"]
	return hasKind || hasMetadata
}

func resourceFromObject(object map[string]any) (Resource, error) {
	apiVersion, ok := object["apiVersion"].(string)
	if !ok || apiVersion == "" {
		return Resource{}, fmt.Errorf("missing apiVersion")
	}
	kind, ok := object["kind"].(string)
	if !ok || kind == "" {
		return Resource{}, fmt.Errorf("missing kind")
	}
	metadata, ok := object["metadata"].(map[string]any)
	if !ok {
		return Resource{}, fmt.Errorf("missing metadata")
	}
	name, ok := metadata["name"].(string)
	if !ok || name == "" {
		return Resource{}, fmt.Errorf("missing metadata.name")
	}
	namespace, _ := metadata["namespace"].(string)

	return Resource{
		APIVersion: apiVersion,
		Kind:       kind,
		Namespace:  namespace,
		Name:       name,
		Object:     object,
	}, nil
}

func isYAML(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".yaml" || extension == ".yml"
}
