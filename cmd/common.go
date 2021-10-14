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
		isSuccess, err := uiService.Decrypt(password)
		if err != nil {
			log.Fatal(err)
		}
		if !isSuccess {
			attempt--
			log.Println("Decryption failed, remaining attempts ", attempt)
			continue
		}
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
