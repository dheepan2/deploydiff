package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/dheepan2/deploydiff/internal/diff"
	"github.com/dheepan2/deploydiff/internal/manifest"
	"github.com/dheepan2/deploydiff/internal/resource"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newCompareCmd(out io.Writer, outputFormat func() string) *cobra.Command {
	var base string
	var head string

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
				return fmt.Errorf("Git reference comparison is not implemented yet; provide two manifest paths")
			}

			before, err := loadState(args[0], "before")
			if err != nil {
				return err
			}
			after, err := loadState(args[1], "after")
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
	return compareCmd
}

func loadState(path, name string) ([]resource.Resource, error) {
	manifests, err := manifest.Load(path)
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
