package cmd

import (
	"bufio"
	"fmt"
	"github.com/CryoCodec/jim/config"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/CryoCodec/jim/files"
	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Performs a system check and gives hints on how to make jim operational.",
	Long:  `Performs a system check and gives hints on how to make jim operational.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()

		fmt.Println("Checking if the config directory ~/.jim is available")
		jimDir := files.GetJimConfigDir()
		if _, err := os.Stat(jimDir); os.IsNotExist(err) {
			fmt.Println("Directory does not exist, creating it...")
			err := os.Mkdir(jimDir, 0740)
			if err != nil {
				dief("Failed to create jim's config directory %s", jimDir)
			} else {
				fmt.Println("---> Success")
			}
		} else {
			fmt.Println("---> Success")
		}

		fmt.Println("Checking config file location")
		path := os.Getenv("JIM_CONFIG_FILE") // env variable has highest priority
		if path == "" {
			fmt.Println("The environment variable JIM_CONFIG_FILE is not set, will use default config location ~/.jim/config.json.enc")
			configFile := filepath.Join(jimDir, "config.json.enc")
			if !files.Exists(configFile) {
				fmt.Println("The config file does not exist. I will create the dummy file config.json for you. Please update the file and use 'jim encrypt' afterwards.")
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
					die("Failed to deserialize json. This is an implementation bug!")
				}

				if files.Exists(dummyFilePath) {
					fmt.Printf("The destination path %s already exists, overwrite? (y/n) \n", dummyFilePath)
					reader := bufio.NewReader(os.Stdin)
					yes, _ := reader.ReadString('\n')
					if strings.TrimSpace(yes) == "y" {
						err := ioutil.WriteFile(dummyFilePath, json, 0740)
						if err != nil {
							dief("Failed to write the demo file at %s: %s", dummyFilePath, err)
						}
					}
				} else {
					err := ioutil.WriteFile(dummyFilePath, json, 0740)
					if err != nil {
						dief("Failed to write the demo file at %s: %s", dummyFilePath, err)
					}
				}

			}
		} else {
			fmt.Printf("Environment variable JIM_CONFIG_FILE is set, using the path %s \n", path)
			if !files.Exists(path) {
				fmt.Println("The configured path does not point to an existing file. Please update the environment variable JIM_CONFIG_FILE.")
			}
		}

		commandExists("pgrep")
		commandExists("sshpass")
		commandExists("ssh")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func commandExists(cmd string) {
	fmt.Printf("Checking if '%s' is available on Path \n", cmd)
	_, err := exec.LookPath(cmd)
	if err == nil {
		fmt.Println("---> Success")
	} else {
		fmt.Printf("Command '%s' could not be found, but is necessary for jim to work. Please install it and make it available on the PATH \n", cmd)
	}
}
