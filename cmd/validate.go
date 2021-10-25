package cmd

import (
	"github.com/CryoCodec/jim/config"
	"github.com/CryoCodec/jim/files"
	"github.com/spf13/cobra"
	"io/ioutil"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate /path/to/file",
	Short: "Checks whether the given file can be parsed as jim config.",
	Long: `Checks whether the given file can be parsed as jim config. 
	The file must be available in plain text. This operation does not work on the encrypted config file.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()

		if !files.Exists(args[0]) {
			die("The passed file does not exist or is a directory")
		}

		fileContents, err := ioutil.ReadFile(args[0])
		if err != nil {
			dief("Error reading file: %s", err)
		}

		_, err = config.UnmarshalJimConfig(fileContents)
		if err != nil {
			dief("The given file could not be read as jim config, reason: %s", err)
		} else {
			die("Congrats, the config file is valid")
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
