package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/services"
	"github.com/spf13/cobra"
)

// reloadCmd represents the reload command
var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reloads the configuration file",
	Long:  `Reloads the configuration file. This is necessary, after the configuration file was changed. After reloading the master password has to be entered again.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()

		uiService := services.NewUiService()
		defer uiService.ShutDown()
		err := uiService.ReloadConfigFile()
		if err == nil {
			fmt.Printf("---> Success")
		} else {
			fmt.Printf("error: %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(reloadCmd)
}
