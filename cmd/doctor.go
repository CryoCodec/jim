package cmd

import (
	"bufio"
	"fmt"
	"github.com/CryoCodec/jim/config"
	"github.com/CryoCodec/jim/files"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/theckman/yacspin"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Performs a system check and gives hints on how to make jim operational.",
	Long:  `Performs a system check and gives hints on how to make jim operational.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()

		var messagesPerStep []string

		spinner, err := CreateSpinner()
		if err != nil {
			log.Debugf("cannot use spinner due to error: %s", err)
		}
		updateSpinnerPrefix(spinner, "Preparing configuration directory")
		spinner.Start()
		messagesPerStep = append(messagesPerStep, yellow("test: configuration directory exists"))
		jimDir := files.GetJimConfigDir()
		if _, err := os.Stat(jimDir); os.IsNotExist(err) {
			err := os.Mkdir(jimDir, 0700)
			if err != nil {
				spinner.StopFail()
				messagesPerStep = append(messagesPerStep, red("Failed to create jim's config directory %s, reason: %s", jimDir, err.Error()))
				printStepMessagesAndDie(messagesPerStep)
			}
			messagesPerStep = append(messagesPerStep, green("created configuration directory ~/.jim"))

		} else {
			messagesPerStep = append(messagesPerStep, green("configuration directory exists"))
		}

		messagesPerStep = append(messagesPerStep, yellow("test: configuration directory is writable"))
		if isWritable(jimDir) {
			messagesPerStep = append(messagesPerStep, green("configuration directory is writable"))
		} else {
			spinner.StopFail()
			messagesPerStep = append(messagesPerStep, red("configuration directory is %s not writable", jimDir))
			printStepMessagesAndDie(messagesPerStep)
		}
		spinner.Stop()
		printStepMessages(messagesPerStep)
		fmt.Println()
		messagesPerStep = nil

		updateSpinnerPrefix(spinner, "Checking config file")
		spinner.Start()
		messagesPerStep = append(messagesPerStep, yellow("test: JIM_CONFIG_FILE environment variable is set"))
		path := os.Getenv("JIM_CONFIG_FILE") // env variable has highest priority
		if path == "" {
			messagesPerStep = append(messagesPerStep, "The environment variable JIM_CONFIG_FILE is not set, will use default config location ~/.jim/config.json.enc")
			configFile := filepath.Join(jimDir, "config.json.enc")
			messagesPerStep = append(messagesPerStep, yellow("test: config file exists"))
			if !files.Exists(configFile) {
				messagesPerStep = append(messagesPerStep, red("The config file does not exist. I will create the dummy file config.json for you. Please update the file and use 'jim encrypt' afterwards."))
				dummyFilePath := filepath.Join(jimDir, "config.json")
				dummyValue := config.JimConfigElement{
					Group: "This is just used for display purposes",
					Env:   "This is just used for display purposes",
					Tag:   "This string is used in the connect command",
					Server: config.JimConfigEntry{
						Host:     "Host name of your Server to connect to",
						Port:     "The SSH Port on the remote server",
						Dir:      "The directory you'd like to start after SSH login",
						Username: "The Username used for authentication",
						Password: "The Password used for authentication",
					},
				}
				jimConfig := config.JimConfig([]config.JimConfigElement{dummyValue})
				json, err := jimConfig.Marshal()
				if err != nil {
					spinner.StopFail()
					messagesPerStep = append(messagesPerStep, red("Failed to deserialize json. This is an implementation bug!"))
					printStepMessagesAndDie(messagesPerStep)
				}

				if files.Exists(dummyFilePath) {
					spinner.Stop()
					printStepMessages(messagesPerStep)
					messagesPerStep = nil
					fmt.Printf("The destination path %s already exists, overwrite? (y/n) \n", dummyFilePath)
					reader := bufio.NewReader(os.Stdin)
					yes, _ := reader.ReadString('\n')
					if strings.TrimSpace(yes) == "y" {
						err := ioutil.WriteFile(dummyFilePath, json, 0700)
						if err != nil {
							die(red("Failed to write the demo file at %s. Reason: %s", dummyFilePath, err))
						}
					}
				} else {
					err := ioutil.WriteFile(dummyFilePath, json, 0700)
					if err != nil {
						spinner.StopFail()
						messagesPerStep = append(messagesPerStep, red("Failed to write the demo file at %s. Reason:  %s", dummyFilePath, err))
						printStepMessagesAndDie(messagesPerStep)
					}
				}
			} else {
				messagesPerStep = append(messagesPerStep, green("config file exists"))
			}
		} else {
			messagesPerStep = append(messagesPerStep, fmt.Sprintf("Environment variable JIM_CONFIG_FILE is set, using the path %s", path))
			messagesPerStep = append(messagesPerStep, yellow("test: config file exists"))
			if !files.Exists(path) {
				spinner.StopFail()
				messagesPerStep = append(messagesPerStep, red("The configured path does not point to an existing file. Please update the environment variable JIM_CONFIG_FILE."))
				printStepMessagesAndDie(messagesPerStep)
			} else {
				messagesPerStep = append(messagesPerStep, green("config file exists"))
			}
		}
		// only required in the special case, when the dummy file is created and the target path already exists
		if !(spinner.Status() == yacspin.SpinnerStopped || spinner.Status() == yacspin.SpinnerStopping) {
			spinner.Stop()
			printStepMessages(messagesPerStep)
		}
		fmt.Println()
		messagesPerStep = nil

		updateSpinnerPrefix(spinner, "Checking required utilities")
		spinner.Start()

		messagesPerStep, ok1 := commandExists("pgrep", messagesPerStep)
		messagesPerStep, ok2 := commandExists("sshpass", messagesPerStep)
		messagesPerStep, ok3 := commandExists("ssh", messagesPerStep)

		if ok1 && ok2 && ok3 {
			spinner.Stop()
			printStepMessages(messagesPerStep)
		} else {
			spinner.StopFail()
			printStepMessagesAndDie(messagesPerStep)
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func commandExists(cmd string, msgs []string) ([]string, bool) {
	msgs = append(msgs, yellow("test: '%s' is available on Path", cmd))
	_, err := exec.LookPath(cmd)
	if err == nil {
		msgs = append(msgs, green("%s exists", cmd))
		return msgs, true
	} else {
		msgs = append(msgs, red("Command '%s' could not be found, but is necessary for jim to work. Please install it and make it available on the PATH", cmd))
		return msgs, false
	}
}

func printStepMessages(msgs []string) {
	for _, msg := range msgs {
		fmt.Printf("    - %s\n", msg)
	}
}

func printStepMessagesAndDie(msgs []string) {
	for _, msg := range msgs {
		fmt.Printf("    - %s\n", msg)
	}
	os.Exit(1)
}

func isWritable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}
