package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time for release binaries.
var Version = "dev"

type options struct {
	configPath string
	verbose    bool
	output     string
}

// Execute runs the DeployDiff command-line interface.
func Execute() {
	if err := NewRootCmd(os.Stdout, os.Stderr).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// NewRootCmd creates the DeployDiff command tree. Separate output writers make
// the CLI straightforward to embed and test.
func NewRootCmd(out, errOut io.Writer) *cobra.Command {
	opts := &options{}
	logger := log.New(errOut, "deploydiff: ", 0)

	rootCmd := &cobra.Command{
		Use:   "deploydiff",
		Short: "Understand Kubernetes deployment changes before they happen.",
		Long: `DeployDiff analyzes Kubernetes deployment manifests,
builds a dependency graph,
compares deployment states,
and explains the impact of infrastructure changes.

It supports Kubernetes YAML, Helm, and Skaffold projects.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if !validOutput(opts.output) {
				return fmt.Errorf("unsupported output format %q (supported: table, json, yaml)", opts.output)
			}
			if opts.verbose {
				logger.Printf("command=%s output=%s config=%q", cmd.CommandPath(), opts.output, opts.configPath)
			}
			return nil
		},
	}

	rootCmd.SetOut(out)
	rootCmd.SetErr(errOut)
	rootCmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVarP(&opts.output, "output", "o", "table", "Output format (table,json,yaml)")

	rootCmd.AddCommand(newCompareCmd(out, func() string { return opts.output }), newVersionCmd(out))
	return rootCmd
}

func validOutput(format string) bool {
	return format == "table" || format == "json" || format == "yaml"
}
