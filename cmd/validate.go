package cmd

import (
	"io/ioutil"
	"log"

	"github.com/CryoCodec/jim/files"
	"github.com/CryoCodec/jim/model"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate /path/to/file",
	Short: "Checks whether the given file can be parsed as jim config.",
	Long: `Checks whether the given file can be parsed as jim config. 
	The file must be available in plain text. This operation does not work on the encrypted config file.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !files.Exists(args[0]) {
			log.Fatal("The passed file does not exist or is a directory")
		}

		fileContents, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Fatal("Error reading file: ", err.Error())
		}

		_, err = model.UnmarshalJimConfig(fileContents)
		if err != nil {
			log.Fatal("The given file could not be read as jim config, reason: ", err.Error())
		} else {
			log.Fatal("Congrats, the config file is valid")
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// validateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// validateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
