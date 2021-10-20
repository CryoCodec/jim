package cmd

import (
	"github.com/CryoCodec/jim/core/services"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"syscall"
)

func requestPWandDecrypt(uiService services.UiService) {
	attempt := 3
	for {
		if attempt == 0 {
			log.Fatal("No more attempts left, exiting.")
		}

		password := readPasswordFromTerminal()
		err := uiService.Decrypt(password)
		if err != nil {
			attempt--
			log.Printf("Decryption failed, reason: %s remaining attempts %d", err, attempt)
			continue
		}

		return
	}
}

func readPasswordFromTerminal() []byte {
	log.Println("Enter master password:")
	bytePassword, err := terminal.ReadPassword(syscall.Stdin)
	if err != nil {
		log.Fatal("Error reading the password from terminal")
	}
	return bytePassword
}

func runPreamble(uiService services.UiService) error {
	for {
		serverState, err := uiService.GetState()
		if err != nil {
			return err
		}

		if serverState.IsReady() {
			return nil
		}

		if serverState.RequiresDecryption() {
			requestPWandDecrypt(uiService)
		}

		if serverState.RequiresConfigFile() {
			if err := uiService.ReloadConfigFile(); err != nil {
				return err
			}
		}
	}
}
