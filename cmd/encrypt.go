package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"syscall"

	b64 "encoding/base64"

	"github.com/CryoCodec/jim/crypto"
	"github.com/CryoCodec/jim/files"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// encryptCmd represents the encrypt command
var encryptCmd = &cobra.Command{
	Use:   "encrypt path/to/file",
	Short: "Encrypts the file at given path, so it can be used with jim",
	Long: `Encrypts the file at path/to/file with a master password. 
	The file may then be used with jim`,
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

		fmt.Println("Enter master password:")
		password, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			die("Error reading the password from terminal. Try again.")
		}

		cipherText, err := crypto.Encrypt(password, fileContents)
		if err != nil {
			dief("Failed to encrypt the given content. Reason: %s", err)
		}

		sEnc := b64.StdEncoding.EncodeToString(cipherText)
		destinationPath := args[0] + ".enc"

		if files.Exists(destinationPath) {
			fmt.Printf("The destination path %s already exists, overwrite? (y/n) \n", destinationPath)
			reader := bufio.NewReader(os.Stdin)
			yes, _ := reader.ReadString('\n')
			if strings.TrimSpace(yes) != "y" {
				return
			}
		}

		err = ioutil.WriteFile(destinationPath, []byte(sEnc), 0644)
		if err != nil {
			dief("Failed to write to %s: %s", destinationPath, err)
		}
		fmt.Printf("Wrote output to %s", destinationPath)
	},
}

func init() {
	rootCmd.AddCommand(encryptCmd)
}
