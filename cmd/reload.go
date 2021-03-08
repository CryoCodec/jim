package cmd

import (
	jim "github.com/CryoCodec/jim/ipc"
	"github.com/spf13/cobra"
)

// reloadCmd represents the reload command
var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reloads the configuration file",
	Long:  `Reloads the configuration file. This is necessary, after the configuration file was changed. After reloading the master password has to be entered again.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		client := jim.CreateClient()
		defer client.Close()
		propagationChan := make(chan jim.Message)
		go jim.ReadMessage(client, propagationChan, Verbose)
		jim.LoadConfigFile(client, propagationChan)
	},
}

func init() {
	rootCmd.AddCommand(reloadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// reloadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// reloadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
