package ipc

import (
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/CryoCodec/jim/files"
	"github.com/CryoCodec/jim/model"
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
		case ResRequireConfigFile:
			loadConfigFile(client, propagationChan)
		case ResNeedDecryption:
			requestPWandDecrypt(client, propagationChan)
		case ResReadyToServe:
			return true
		default:
			log.Fatal(fmt.Sprintf("Received unexpected message %s, when requesting entries.", msgCodeToString[uint16(message.Code)]))
		}
	}
}

func ListEntries(client *ipc.Client, propagationChan chan Message) model.ListResponse {
	for {
		writetoServer(client, ReqListEntries, []byte{})
		message := <-propagationChan
		switch message.Code {
		case ResListEntries:
			result, err := model.UnmarshalListResponse(message.Payload)
			if err != nil {
				log.Fatal("Failed to deserialize json response. This is likely an implementation bug. Reason: ", err.Error())
			}
			return result
		case ResNeedDecryption:
			log.Fatal("Server was in wrong state. This is likely an implementation bug.")
		case ResError:
			log.Fatal("Server error: ", string(message.Payload))
		default:
			log.Fatal(fmt.Sprintf("Received unexpected message %s, when requesting entries.", msgCodeToString[uint16(message.Code)]))
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
		case ResJsonDeserializationFailed:
			log.Fatal(fmt.Sprintf("Config file is corrupted. Could not unmarshal json. Please correct your config file. Error: %s", string(response.Payload)))
		default:
			log.Fatal(fmt.Sprintf("Received unexpected message %s, when attempting decryption. Error: %s", msgCodeToString[uint16(response.Code)], string(response.Payload)))
		}
	}
}

func loadConfigFile(client *ipc.Client, propagationChan chan Message) {
	path := files.GetJimConfigPath()
	writetoServer(client, ReqLoadFile, []byte(path))
	switch response := <-propagationChan; response.Code {
	case ResError:
		log.Fatal(fmt.Sprintf("Server failed to load config file, reason: %s", string(response.Payload)))
	case ResSuccess:
		return
	default:
		log.Fatal(fmt.Sprintf("Received unexpected message %s, when loading config file", msgCodeToString[uint16(response.Code)]))
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
