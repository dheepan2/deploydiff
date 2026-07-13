package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newCompareCmd(out io.Writer) *cobra.Command {
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
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("comparison engine is not implemented yet")
		},
	}

	compareCmd.SetOut(out)
	compareCmd.Flags().StringVar(&base, "base", "", "Base Git reference")
	compareCmd.Flags().StringVar(&head, "head", "", "Head Git reference")
	return compareCmd
}
