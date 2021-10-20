package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints jim's version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("jim version 1.0.0-rc")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
