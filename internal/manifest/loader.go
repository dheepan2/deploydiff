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
// alone is not enough because Helm Chart.yaml files use that field too.
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
	reader, closeReader, err := manifestReader(path, discover)
	if err != nil {
		return nil, err
	}
	if closeReader != nil {
		defer closeReader()
	}
	if reader == nil {
		return nil, nil
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

func manifestReader(path string, discover bool) (io.Reader, func() error, error) {
	if discover {
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("read manifest file %q: %w", path, err)
		}
		if hasTemplateActions(contents) {
			return nil, nil, nil
		}
		return bytes.NewReader(contents), nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open manifest file %q: %w", path, err)
	}
	return file, file.Close, nil
}

func hasTemplateActions(contents []byte) bool {
	return bytes.Contains(contents, []byte("{{")) && bytes.Contains(contents, []byte("}}"))
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
