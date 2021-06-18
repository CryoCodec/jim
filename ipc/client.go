package ipc

import (
	"encoding/json"
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	"log"
	"syscall"
	"time"

	"github.com/CryoCodec/jim/files"
	ipc "github.com/james-barrow/golang-ipc"
	"golang.org/x/crypto/ssh/terminal"
)

type IpcClient struct {
	cc              *ipc.Client
	propagationChan chan Message
}

// InitializeClient creates an ipc client, which may be used to
// write and receive data from a unix domain socket/named pipe
func InitializeClient(logVerbose bool) IpcClient {
	config := &ipc.ClientConfig{
		Timeout: 2,
	}
	cc, err := ipc.StartClient("jimssocket", config)
	if err != nil {
		log.Fatal("Could not create ipc client. Reason:", err)
	}
	msgChannel := startReceiving(cc, logVerbose)
	return IpcClient{cc, msgChannel}
}

func (ipcClient *IpcClient) Close() {
	ipcClient.cc.Close()
}

// MakeServerReady ensures the server is in the correct state ready and decrypted.
// If necessary this method will cause the server to load the config file and ask the user
// to enter the master password for decryption.
func (ipcClient *IpcClient) MakeServerReady() bool {
	for {
		ipcClient.writetoServer(ReqStatus, []byte{})
		message := <-ipcClient.propagationChan
		switch message.Code {
		case ResRequireConfigFile:
			ipcClient.LoadConfigFile()
		case ResNeedDecryption:
			ipcClient.requestPWandDecrypt()
		case ResReadyToServe:
			return true
		default:
			die(fmt.Sprintf("Received unexpected message %s, when requesting entries.", msgCodeToString[uint16(message.Code)]), ipcClient.cc)
		}
	}
}

// IsServerReady checks whether the server is ready to serve
func (ipcClient *IpcClient) IsServerReady() bool {
	for {
		ipcClient.writetoServer(ReqStatus, []byte{})
		message := <-ipcClient.propagationChan
		switch message.Code {
		case ResReadyToServe:
			return true
		default:
			return false
		}
	}
}

// MatchClosestN gets a list of potentially matching entries in the config file
func (ipcClient *IpcClient) MatchClosestN(query string) []string {
	for {
		ipcClient.writetoServer(ReqClosestN, []byte(query))
		message := <-ipcClient.propagationChan
		switch message.Code {
		case ResClosestN:
			var arr []string
			err := json.Unmarshal(message.Payload, &arr)
			if err != nil {
				return []string{}
			}
			return arr
		default: // since this is only used for shell completions, we return an empty array in all other cases.
			return []string{}
		}
	}
}

// ListEntries asks the server for all entries in the config file and returns these.
// The server has to be in ready state.
func (ipcClient *IpcClient) ListEntries() chan domain.ListResponseElement {
	out := make(chan domain.ListResponseElement)
	ipcClient.writetoServer(ReqListEntries, []byte{})
	go func() {
		for {
			message := <-ipcClient.propagationChan
			switch message.Code {
			case ResListEntries:
				result, err := domain.UnmarshalListResponseElement(message.Payload)
				if err != nil {
					die(fmt.Sprintf("Failed to deserialize json response. This is likely an implementation bug. Reason: %s", err.Error()), ipcClient.cc)
				}
				out <- result
			case ResNeedDecryption:
				die("Server was in wrong state. This is likely an implementation bug.", ipcClient.cc)
			case ResError:
				die(fmt.Sprintf("Server error: %s", string(message.Payload)), ipcClient.cc)
			case ResSuccess:
				close(out)
				return
			default:
				die(fmt.Sprintf("Received unexpected message %s, when requesting entries.", msgCodeToString[uint16(message.Code)]), ipcClient.cc)
			}
		}
	}()
	return out
}

// GetMatchingServer asks the server for a matching entry for the query string.
// The server has to be in ready state.
func (ipcClient *IpcClient) GetMatchingServer(query string) domain.MatchResponse {
	ipcClient.writetoServer(ReqClosestMatch, []byte(query))
	message := <-ipcClient.propagationChan
	switch message.Code {
	case ResClosestMatch:
		result, err := domain.UnmarshalMatchResponse(message.Payload)
		if err != nil {
			die(fmt.Sprintf("Failed to deserialize json response. This is likely an implementation bug. Reason: %s", err.Error()), ipcClient.cc)
		}
		return result
	case ResNoMatch:
		die("No Server matched your query.", ipcClient.cc)
	case ResError:
		die(fmt.Sprintf("Server error: %s", string(message.Payload)), ipcClient.cc)
	default:
		die(fmt.Sprintf("Received unexpected message %s, when requesting entries.", msgCodeToString[uint16(message.Code)]), ipcClient.cc)
	}
	panic("reached unreachable code. ( Well, wasn't so unreachable after all, hu? )")
}

// startReceiving will read from the socket in go routine until forever. Domain specific messages are forwarded via the returned channel.
func startReceiving(client *ipc.Client, verbose bool) chan Message {
	propagationChan := make(chan Message)

	go func() {
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
	}()

	return propagationChan
}

func (ipcClient *IpcClient) requestPWandDecrypt() {
	attempt := 3
	for {
		if attempt == 0 {
			die("No more attempts left, exiting.", ipcClient.cc)
		}
		log.Println("Enter master password:")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Println("Error reading the password from terminal")
			attempt--
			continue
		}
		ipcClient.writetoServer(ReqAttemptDecryption, bytePassword)
		switch response := <-ipcClient.propagationChan; response.Code {
		case ResDecryptionFailed:
			attempt--
			log.Println("Decryption failed, remaining attempts ", attempt)
			continue
		case ResSuccess:
			return
		case ResJsonDeserializationFailed:
			die(fmt.Sprintf("Config file is corrupted. Could not unmarshal json. Please correct your config file. Error: %s", string(response.Payload)), ipcClient.cc)
		default:
			die(fmt.Sprintf("Received unexpected message %s, when attempting decryption. Error: %s", msgCodeToString[uint16(response.Code)], string(response.Payload)), ipcClient.cc)
		}
	}
}

// LoadConfigFile causes the server to load the config file.
func (ipcClient *IpcClient) LoadConfigFile() {
	path, err := files.GetJimConfigFilePath()
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Trying to load config file from %s", path)
	ipcClient.writetoServer(ReqLoadFile, []byte(path))
	switch response := <-ipcClient.propagationChan; response.Code {
	case ResError:
		log.Fatal(fmt.Sprintf("Server failed to load config file, reason: %s", string(response.Payload)))
	case ResSuccess:
		log.Printf("---> Success")
		return
	default:
		die(fmt.Sprintf("Received unexpected message %s, when loading config file", msgCodeToString[uint16(response.Code)]), ipcClient.cc)
	}
}

func (ipcClient *IpcClient) writetoServer(msgType int, message []byte) {
	// sleep until we're connected. ReadMessage will exit the application on timeout, so this is correct.
	client := ipcClient.cc
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
