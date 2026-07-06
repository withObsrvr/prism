package prism

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/withObsrvr/prism/internal/buildinfo"
)

// Set via ldflags at build time:
//
//	go build -ldflags "-X github.com/withObsrvr/prism/cmd/prism.version=1.0.0"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("prism %s (commit: %s, built: %s)\n", buildinfo.Version, buildinfo.Commit, buildinfo.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
