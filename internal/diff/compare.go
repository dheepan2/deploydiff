// Package diff compares normalized Kubernetes deployment states.
package diff

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/dheepan2/deploydiff/internal/resource"
)

// Modification describes a resource with the same identity in both states but
// different manifest contents.
type Modification struct {
	Before  resource.Resource
	After   resource.Resource
	Changes []FieldChange
}

// FieldChange describes one changed field in a resource manifest.
type FieldChange struct {
	Path          string `json:"path" yaml:"path"`
	Before        any    `json:"before" yaml:"before"`
	After         any    `json:"after" yaml:"after"`
	BeforePresent bool   `json:"beforePresent" yaml:"beforePresent"`
	AfterPresent  bool   `json:"afterPresent" yaml:"afterPresent"`
	Sensitive     bool   `json:"sensitive" yaml:"sensitive"`
}

// Result is the deterministic set of changes between two deployment states.
type Result struct {
	Added    []resource.Resource
	Removed  []resource.Resource
	Modified []Modification
}

// HasChanges reports whether the comparison found any resource changes.
func (result Result) HasChanges() bool {
	return len(result.Added) > 0 || len(result.Removed) > 0 || len(result.Modified) > 0
}

// Compare finds resources that were added, removed, or modified between two
// parsed deployment states. Source paths and document numbers do not affect
// modification detection; only the manifest object is compared.
func Compare(before, after []resource.Resource) (Result, error) {
	beforeByID, err := index(before, "before")
	if err != nil {
		return Result{}, err
	}
	afterByID, err := index(after, "after")
	if err != nil {
		return Result{}, err
	}

	result := Result{}
	for id, beforeResource := range beforeByID {
		afterResource, exists := afterByID[id]
		if !exists {
			result.Removed = append(result.Removed, beforeResource)
			continue
		}
		changes := fieldChanges(beforeResource.Object, afterResource.Object)
		if len(changes) > 0 {
			redactSensitiveChanges(beforeResource.ID.Kind, changes)
			result.Modified = append(result.Modified, Modification{Before: beforeResource, After: afterResource, Changes: changes})
		}
	}
	for id, afterResource := range afterByID {
		if _, exists := beforeByID[id]; !exists {
			result.Added = append(result.Added, afterResource)
		}
	}

	sortResources(result.Added)
	sortResources(result.Removed)
	sort.Slice(result.Modified, func(i, j int) bool {
		return result.Modified[i].After.ID.String() < result.Modified[j].After.ID.String()
	})
	return result, nil
}

func redactSensitiveChanges(kind string, changes []FieldChange) {
	if kind != "Secret" {
		return
	}
	for index := range changes {
		if !isSecretDataPath(changes[index].Path) {
			continue
		}
		changes[index].Sensitive = true
		if changes[index].BeforePresent {
			changes[index].Before = "<redacted>"
		}
		if changes[index].AfterPresent {
			changes[index].After = "<redacted>"
		}
	}
}

func isSecretDataPath(path string) bool {
	return path == "data" || path == "stringData" || hasPathPrefix(path, "data") || hasPathPrefix(path, "stringData")
}

func hasPathPrefix(path, prefix string) bool {
	return len(path) > len(prefix) && path[:len(prefix)] == prefix && (path[len(prefix)] == '.' || path[len(prefix)] == '[')
}

func fieldChanges(before, after map[string]any) []FieldChange {
	changes := []FieldChange{}
	diffValues("", before, after, true, true, &changes)
	return changes
}

func diffValues(path string, before, after any, beforePresent, afterPresent bool, changes *[]FieldChange) {
	if !beforePresent || !afterPresent {
		*changes = append(*changes, FieldChange{Path: path, Before: before, After: after, BeforePresent: beforePresent, AfterPresent: afterPresent})
		return
	}
	if reflect.DeepEqual(before, after) {
		return
	}

	beforeMap, beforeIsMap := before.(map[string]any)
	afterMap, afterIsMap := after.(map[string]any)
	if beforeIsMap && afterIsMap {
		keys := make(map[string]struct{}, len(beforeMap)+len(afterMap))
		for key := range beforeMap {
			keys[key] = struct{}{}
		}
		for key := range afterMap {
			keys[key] = struct{}{}
		}
		sortedKeys := make([]string, 0, len(keys))
		for key := range keys {
			sortedKeys = append(sortedKeys, key)
		}
		sort.Strings(sortedKeys)
		for _, key := range sortedKeys {
			beforeValue, beforeExists := beforeMap[key]
			afterValue, afterExists := afterMap[key]
			diffValues(fieldPath(path, key), beforeValue, afterValue, beforeExists, afterExists, changes)
		}
		return
	}

	beforeList, beforeIsList := before.([]any)
	afterList, afterIsList := after.([]any)
	if beforeIsList && afterIsList {
		length := len(beforeList)
		if len(afterList) > length {
			length = len(afterList)
		}
		for index := 0; index < length; index++ {
			beforeExists := index < len(beforeList)
			afterExists := index < len(afterList)
			var beforeValue, afterValue any
			if beforeExists {
				beforeValue = beforeList[index]
			}
			if afterExists {
				afterValue = afterList[index]
			}
			diffValues(fmt.Sprintf("%s[%d]", path, index), beforeValue, afterValue, beforeExists, afterExists, changes)
		}
		return
	}

	*changes = append(*changes, FieldChange{Path: path, Before: before, After: after, BeforePresent: true, AfterPresent: true})
}

func fieldPath(parent, key string) string {
	if isSimpleFieldName(key) {
		if parent == "" {
			return key
		}
		return parent + "." + key
	}
	return fmt.Sprintf("%s[%q]", parent, key)
}

func isSimpleFieldName(name string) bool {
	if name == "" {
		return false
	}
	for index, character := range name {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || character == '_' || character == '-' || (index > 0 && character >= '0' && character <= '9') {
			continue
		}
		return false
	}
	return true
}

func index(resources []resource.Resource, state string) (map[resource.ID]resource.Resource, error) {
	indexed := make(map[resource.ID]resource.Resource, len(resources))
	for _, current := range resources {
		if previous, exists := indexed[current.ID]; exists {
			return nil, fmt.Errorf("duplicate resource %s in %s state (%s and %s)", current.ID, state, location(previous), location(current))
		}
		indexed[current.ID] = current
	}
	return indexed, nil
}

func sortResources(resources []resource.Resource) {
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ID.String() < resources[j].ID.String()
	})
}

func location(resource resource.Resource) string {
	if resource.Source == "" {
		return "unknown source"
	}
	if resource.Document == 0 {
		return resource.Source
	}
	return fmt.Sprintf("%s document %d", resource.Source, resource.Document)
}
