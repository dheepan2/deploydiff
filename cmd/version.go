package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newVersionCmd(out io.Writer) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print DeployDiff version",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintln(out, Version)
		},
	}
	versionCmd.SetOut(out)
	return versionCmd
}
