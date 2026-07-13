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
	Before resource.Resource
	After  resource.Resource
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
		if !reflect.DeepEqual(beforeResource.Object, afterResource.Object) {
			result.Modified = append(result.Modified, Modification{Before: beforeResource, After: afterResource})
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
