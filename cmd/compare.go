package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dheepan2/deploydiff/internal/diff"
	"github.com/dheepan2/deploydiff/internal/gitref"
	"github.com/dheepan2/deploydiff/internal/manifest"
	"github.com/dheepan2/deploydiff/internal/resource"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newCompareCmd(out io.Writer, outputFormat func() string) *cobra.Command {
	var base string
	var head string
	var discover bool
	var manifestPath string

	compareCmd := &cobra.Command{
		Use:   "compare [before] [after]",
		Short: "Compare two Kubernetes deployment states",
		Long: `Compare Kubernetes deployment manifests and explain
the impact of changes before deployment.

Provide either two manifest paths or both --base and --head Git references.`,
		Args: func(cmd *cobra.Command, args []string) error {
			hasGitRefs := cmd.Flags().Changed("base") || cmd.Flags().Changed("head")
			switch {
			case len(args) == 2 && !hasGitRefs:
				return nil
			case len(args) == 0 && base != "" && head != "":
				return nil
			case len(args) == 0 && hasGitRefs:
				return fmt.Errorf("both --base and --head are required together")
			default:
				return fmt.Errorf("provide two manifest paths or both --base and --head")
			}
		},
		RunE: func(_ *cobra.Command, args []string) error {
			if base != "" || head != "" {
				return compareGitReferences(out, outputFormat(), base, head, manifestPath)
			}

			before, err := loadState(args[0], "before", discover)
			if err != nil {
				return err
			}
			after, err := loadState(args[1], "after", discover)
			if err != nil {
				return err
			}
			result, err := diff.Compare(before, after)
			if err != nil {
				return fmt.Errorf("compare deployment states: %w", err)
			}
			return renderComparison(out, outputFormat(), result)
		},
	}

	compareCmd.SetOut(out)
	compareCmd.Flags().StringVar(&base, "base", "", "Base Git reference")
	compareCmd.Flags().StringVar(&head, "head", "", "Head Git reference")
	compareCmd.Flags().StringVar(&manifestPath, "path", ".", "Manifest path within each Git reference")
	compareCmd.Flags().BoolVar(&discover, "discover", false, "Discover Kubernetes manifests in YAML files and ignore unrelated YAML")
	return compareCmd
}

func compareGitReferences(out io.Writer, outputFormat, base, head, manifestPath string) error {
	manifestPath, err := safeGitManifestPath(manifestPath)
	if err != nil {
		return err
	}
	temporaryDirectory, err := os.MkdirTemp("", "deploydiff-")
	if err != nil {
		return fmt.Errorf("create temporary Git comparison directory: %w", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	beforePath := filepath.Join(temporaryDirectory, "before")
	afterPath := filepath.Join(temporaryDirectory, "after")
	if err := gitref.Export(base, beforePath); err != nil {
		return fmt.Errorf("materialize base Git reference: %w", err)
	}
	if err := gitref.Export(head, afterPath); err != nil {
		return fmt.Errorf("materialize head Git reference: %w", err)
	}

	before, err := loadGitState(filepath.Join(beforePath, manifestPath), "base")
	if err != nil {
		return err
	}
	after, err := loadGitState(filepath.Join(afterPath, manifestPath), "head")
	if err != nil {
		return err
	}
	result, err := diff.Compare(before, after)
	if err != nil {
		return fmt.Errorf("compare deployment states: %w", err)
	}
	return renderComparison(out, outputFormat, result)
}

func safeGitManifestPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("Git manifest path must stay within the repository: %q", path)
	}
	return cleanPath, nil
}

func loadGitState(path, name string) ([]resource.Resource, error) {
	resources, err := loadState(path, name, true)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return resources, err
}

func loadState(path, name string, discover bool) ([]resource.Resource, error) {
	var manifests []manifest.Resource
	var err error
	if discover {
		manifests, err = manifest.Discover(path)
	} else {
		manifests, err = manifest.Load(path)
	}
	if err != nil {
		return nil, fmt.Errorf("load %s deployment state: %w", name, err)
	}
	resources, err := resource.ParseAll(manifests)
	if err != nil {
		return nil, fmt.Errorf("parse %s deployment state: %w", name, err)
	}
	return resources, nil
}

type comparisonReport struct {
	Added    []string         `json:"added" yaml:"added"`
	Removed  []string         `json:"removed" yaml:"removed"`
	Modified []modifiedReport `json:"modified" yaml:"modified"`
}

type modifiedReport struct {
	Resource string             `json:"resource" yaml:"resource"`
	Changes  []diff.FieldChange `json:"changes" yaml:"changes"`
}

func renderComparison(out io.Writer, format string, result diff.Result) error {
	report := comparisonReport{
		Added:    resourceIDs(result.Added),
		Removed:  resourceIDs(result.Removed),
		Modified: modifiedReports(result.Modified),
	}

	switch format {
	case "table":
		renderTable(out, report)
		return nil
	case "json":
		encoded, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("render JSON comparison: %w", err)
		}
		_, err = fmt.Fprintln(out, string(encoded))
		return err
	case "yaml":
		encoded, err := yaml.Marshal(report)
		if err != nil {
			return fmt.Errorf("render YAML comparison: %w", err)
		}
		_, err = out.Write(encoded)
		return err
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func renderTable(out io.Writer, report comparisonReport) {
	if len(report.Added) == 0 && len(report.Removed) == 0 && len(report.Modified) == 0 {
		fmt.Fprintln(out, "No deployment changes.")
		return
	}

	fmt.Fprintln(out, "Deployment Comparison")
	renderTableSection(out, "Added", "+", report.Added)
	renderTableSection(out, "Removed", "-", report.Removed)
	renderModifiedTableSection(out, report.Modified)
}

func renderTableSection(out io.Writer, title, marker string, resources []string) {
	if len(resources) == 0 {
		return
	}
	fmt.Fprintf(out, "\n%s (%d)\n", title, len(resources))
	for _, resource := range resources {
		fmt.Fprintf(out, "%s %s\n", marker, resource)
	}
}

func renderModifiedTableSection(out io.Writer, modifications []modifiedReport) {
	if len(modifications) == 0 {
		return
	}
	fmt.Fprintf(out, "\nModified (%d)\n", len(modifications))
	for _, modification := range modifications {
		fmt.Fprintf(out, "~ %s\n", modification.Resource)
		for _, change := range modification.Changes {
			fmt.Fprintf(out, "  %s: %s → %s\n", change.Path, formatFieldValue(change.Before, change.BeforePresent), formatFieldValue(change.After, change.AfterPresent))
		}
	}
}

func formatFieldValue(value any, present bool) string {
	if !present {
		return "<absent>"
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}

func resourceIDs(resources []resource.Resource) []string {
	ids := make([]string, len(resources))
	for i, resource := range resources {
		ids[i] = resource.ID.String()
	}
	return ids
}

func modifiedReports(modifications []diff.Modification) []modifiedReport {
	reports := make([]modifiedReport, len(modifications))
	for i, modification := range modifications {
		reports[i] = modifiedReport{Resource: modification.After.ID.String(), Changes: modification.Changes}
	}
	return reports
}
