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

// CreateClient creates an ipc client, which may be used to
// write and receive data from a unix domain socket/named pipe
func CreateClient() *ipc.Client {
	config := &ipc.ClientConfig{
		Timeout: 2,
	}
	cc, err := ipc.StartClient("jimssocket", config)
	if err != nil {
		log.Fatal("Could not create ipc client. Reason:", err)
	}

	return cc
}

// IsServerStatusReady ensures the server is in the correct state ready and decrypted.
// If necessary this method will cause the server to load the config file and ask the user
// to enter the master password for decryption.
func IsServerStatusReady(client *ipc.Client, propagationChan chan Message) bool {
	for {
		writetoServer(client, ReqStatus, []byte{})
		message := <-propagationChan
		switch message.Code {
		case ResRequireConfigFile:
			LoadConfigFile(client, propagationChan)
		case ResNeedDecryption:
			requestPWandDecrypt(client, propagationChan)
		case ResReadyToServe:
			return true
		default:
			die(fmt.Sprintf("Received unexpected message %s, when requesting entries.", msgCodeToString[uint16(message.Code)]), client)
		}
	}
}

// ListEntries asks the server for all entries in the config file and returns these.
// The server has to be in ready state.
func ListEntries(client *ipc.Client, propagationChan chan Message) chan model.ListResponseElement {
	out := make(chan model.ListResponseElement)
	writetoServer(client, ReqListEntries, []byte{})
	go func() {
		for {
			message := <-propagationChan
			switch message.Code {
			case ResListEntries:
				result, err := model.UnmarshalListResponseElement(message.Payload)
				if err != nil {
					die(fmt.Sprintf("Failed to deserialize json response. This is likely an implementation bug. Reason: %s", err.Error()), client)
				}
				out <- result
			case ResNeedDecryption:
				die("Server was in wrong state. This is likely an implementation bug.", client)
			case ResError:
				die(fmt.Sprintf("Server error: %s", string(message.Payload)), client)
			case ResSuccess:
				close(out)
				return
			default:
				die(fmt.Sprintf("Received unexpected message %s, when requesting entries.", msgCodeToString[uint16(message.Code)]), client)
			}
		}
	}()
	return out
}

// GetMatchingServer asks the server for a matching entry for the query string.
// The server has to be in ready state.
func GetMatchingServer(query string, client *ipc.Client, propagationChan chan Message) model.MatchResponse {
	writetoServer(client, ReqClosestMatch, []byte(query))
	message := <-propagationChan
	switch message.Code {
	case ResClosestMatch:
		result, err := model.UnmarshalMatchResponse(message.Payload)
		if err != nil {
			die(fmt.Sprintf("Failed to deserialize json response. This is likely an implementation bug. Reason: %s", err.Error()), client)
		}
		return result
	case ResNoMatch:
		die("No Server matched your query.", client)
	case ResError:
		die(fmt.Sprintf("Server error: %s", string(message.Payload)), client)
	default:
		die(fmt.Sprintf("Received unexpected message %s, when requesting entries.", msgCodeToString[uint16(message.Code)]), client)
	}
	panic("reached unreachable code. ( Well, wasn't so unreachable after all, hu? )")
}

// ReadMessage will read from the socket until forever. Domain specific messages are forwarded via the propagationChan.
// Run this function in a Go routine.
func ReadMessage(client *ipc.Client, propagationChan chan Message, verbose bool) {
	errorCounter := 0
	for {
		m, err := client.Read()

		if err != nil {
			if !(err.Error() == "Client has closed the connection") { // this message will always be sent, once we close the client intentionally
				die(fmt.Sprintf("IPC Communication breakdown. Reason: %s ", err.Error()), client)
			}
			return
		}
		switch m.MsgType {
		case -1: // message type -1 is status change and only used internally
			if verbose {
				log.Println("Status update: " + m.Status)
			}
		case -2: // message type -2 is an error, these won't automatically cause the recieve channel to close.
			log.Println("Error: " + err.Error())
			errorCounter++
			if errorCounter > 10 {
				die("Exhausted retry budget, application will exit. Please try again.", client)
			}
			time.Sleep(200 * time.Millisecond)
		default:
			if verbose {
				log.Println("Client received message: " + msgCodeToString[uint16(m.MsgType)])
			}
			propagationChan <- Message{Code(m.MsgType), m.Data}
		}
	}
}

func requestPWandDecrypt(client *ipc.Client, propagationChan chan Message) {
	attempt := 3
	for {
		if attempt == 0 {
			die("No more attempts left, exiting.", client)
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
			attempt--
			log.Println("Decryption failed, remaining attempts ", attempt)
			continue
		case ResSuccess:
			return
		case ResJsonDeserializationFailed:
			die(fmt.Sprintf("Config file is corrupted. Could not unmarshal json. Please correct your config file. Error: %s", string(response.Payload)), client)
		default:
			die(fmt.Sprintf("Received unexpected message %s, when attempting decryption. Error: %s", msgCodeToString[uint16(response.Code)], string(response.Payload)), client)
		}
	}
}

// LoadConfigFile causes the server to load the config file.
func LoadConfigFile(client *ipc.Client, propagationChan chan Message) {
	path, err := files.GetJimConfigFilePath()
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Trying to load config file from %s", path)
	writetoServer(client, ReqLoadFile, []byte(path))
	switch response := <-propagationChan; response.Code {
	case ResError:
		log.Fatal(fmt.Sprintf("Server failed to load config file, reason: %s", string(response.Payload)))
	case ResSuccess:
		log.Printf("---> Success")
		return
	default:
		die(fmt.Sprintf("Received unexpected message %s, when loading config file", msgCodeToString[uint16(response.Code)]), client)
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
		die(fmt.Sprintf("Error writing to server: %s", err.Error()), client)
	}
}

func die(message string, client *ipc.Client) {
	if client != nil {
		client.Close()
	}
	log.Fatal(message)
}
