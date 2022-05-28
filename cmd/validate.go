package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/config"
	"github.com/CryoCodec/jim/files"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"strconv"
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

		spinner, err := CreateSpinner()
		if err != nil {
			log.Debugf("cannot use spinner due to error: %s", err)
		}

		updateSpinnerPrefix(spinner, "Validating")
		spinner.Start()

		jimConf, err := config.UnmarshalJimConfig(fileContents)
		if err != nil {
			die(red("The given file could not be read as jim config, reason: %s", err))
		}

		var validationErrors []validationError
		var messagesPerStep []string

		messagesPerStep = append(messagesPerStep, yellow("Checking for invalid ports"))

		duplicatesMap := make(map[string]int)
		foundInvalidPorts := false
		for _, el := range jimConf {
			// checking for duplicated tags
			duplicatesMap[el.Tag] += 1

			// checking for invalid port numbers
			port, err := strconv.Atoi(el.Server.Port)
			if err != nil {
				foundInvalidPorts = true
				validationErrors = append(validationErrors, validationError{
					tag:    el.Tag,
					reason: "The port must be a numeric value between 0 and 65535",
				})
			}

			if port < 0 || port > 65535 {
				foundInvalidPorts = true
				validationErrors = append(validationErrors, validationError{
					tag:    el.Tag,
					reason: "The port must be a numeric value between 0 and 65535",
				})
			}
		}

		if foundInvalidPorts {
			messagesPerStep = append(messagesPerStep, red("Found invalid ports"))
		} else {
			messagesPerStep = append(messagesPerStep, green("All ports are valid"))
		}

		messagesPerStep = append(messagesPerStep, yellow("Checking for duplicated tags"))
		foundDuplicates := false
		for tag, count := range duplicatesMap {
			if count > 1 {
				foundDuplicates = true
				validationErrors = append(validationErrors, validationError{tag: tag, reason: "Tag is used more than once. Tags should be unique."})
			}
		}

		if foundDuplicates {
			messagesPerStep = append(messagesPerStep, red("Found duplicated tags"))
		} else {
			messagesPerStep = append(messagesPerStep, green("All tags are unique"))
		}

		if foundDuplicates || foundInvalidPorts {
			spinner.StopFail()
			printStepMessages(messagesPerStep)
			fmt.Println()
			printInvalidEntries(validationErrors)
			os.Exit(1)
		} else {
			spinner.Stop()
			printStepMessages(messagesPerStep)
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

type validationError struct {
	tag    string
	reason string
}

func printInvalidEntries(validationErrors []validationError) {
	c := color.New(color.Bold).Add(color.Underline)
	c.Println("These errors were found: ")

	for _, err := range validationErrors {
		fmt.Printf("Tag: %s\n", err.tag)
		fmt.Printf("Reason: %s\n", err.reason)
		fmt.Println()
	}
}
