package ipc

import (
	"fmt"
	"log"
	"syscall"
	"time"

	ipc "github.com/james-barrow/golang-ipc"
	"golang.org/x/crypto/ssh/terminal"
)

type GenericError struct {
	message string
}

func (e *GenericError) Error() string {
	return fmt.Sprintf("Error: %s", e.message)
}

func CreateClient() *ipc.Client {
	config := &ipc.ClientConfig{
		Timeout:    2,
		RetryTimer: 2,
	}
	cc, err := ipc.StartClient("jimssocket", config)
	if err != nil {
		log.Fatal("Could not create ipc client. Reason:", err)
	}

	return cc
}

func IsServerStatusReady(client *ipc.Client, propagationChan chan Message) bool {
	for {
		writetoServer(client, ReqStatus, []byte{})
		message := <-propagationChan
		switch message.Code {
		case ResNeedDecryption:
			requestPWandDecrypt(client, propagationChan)
		case ResReadyToServe:
			return true
		default:
			log.Fatal("Received unexpected message, when requesting server status.")
		}
	}
}

func ReadMessage(client *ipc.Client, propagationChan chan Message) (interface{}, error) {
	errorCounter := 0
	for {
		m, err := client.Read()

		if err != nil {
			log.Fatal("IPC Communication breakdown. Reason: ", err.Error())
		}
		switch m.MsgType {
		case -1: // message type -1 is status change and only used internally
			// once a verbosity flag is implemented, this may print additional information
			log.Println("Status update: " + m.Status)
		case -2: // message type -2 is an error, these won't automatically cause the recieve channel to close.
			log.Println("Error: " + err.Error())
			errorCounter++
			if errorCounter > 10 {
				log.Fatal("Exhausted retry budget, application will exit. Please try again.")
			}
			time.Sleep(200 * time.Millisecond)
		default:
			log.Println("Client received message: " + msgCodeToString[uint16(m.MsgType)] + ": " + string(m.Data))
			propagationChan <- Message{Code(m.MsgType), m.Data}
		}
	}
}

func requestPWandDecrypt(client *ipc.Client, propagationChan chan Message) {
	attempt := 3
	for {
		if attempt == 0 {
			log.Fatal("No more attempts left, exiting.")
		}
		log.Println("Enter master password:")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Println("Error reading the password from terminal")
			attempt--
			continue
		}
		writetoServer(client, ReqAttemptDecryption, bytePassword)
		switch response := <-propagationChan; response.Code {
		case ResDecryptionFailed:
			log.Println("Decryption failed, remaining attempts ", attempt)
			attempt--
			continue
		case ResSuccess:
			return
		default:
			log.Fatal("Client received unexpected message: " + msgCodeToString[uint16(response.Code)] + ": " + string(response.Payload))
		}
	}
}

func writetoServer(client *ipc.Client, msgType int, message []byte) {
	// sleep until we're connected. ReadMessage will exit the application on timeout, so this is correct.
	for {
		if client.Status() != "Connected" {
			time.Sleep(200 * time.Millisecond)
		} else {
			break
		}
	}

	err := client.Write(msgType, message)
	if err != nil {
		log.Fatal("Error writing to server:", err.Error())
	}
}
