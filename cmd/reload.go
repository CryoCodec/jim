package cmd

import (
	"github.com/CryoCodec/jim/core/services"
	"github.com/spf13/cobra"
	"log"
)

// reloadCmd represents the reload command
var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reloads the configuration file",
	Long:  `Reloads the configuration file. This is necessary, after the configuration file was changed. After reloading the master password has to be entered again.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		uiService := services.NewUiService(Verbose)
		defer uiService.ShutDown()
		err := uiService.ReloadConfigFile()
		if err != nil {
			log.Printf("---> Success")
		} else {
			log.Print(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(reloadCmd)
}
