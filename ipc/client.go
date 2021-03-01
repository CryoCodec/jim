package ipc

import (
	"fmt"
	"log"
	"time"

	ipc "github.com/james-barrow/golang-ipc"
)

type GenericError struct {
	message string
}

func (e *GenericError) Error() string {
	return fmt.Sprintf("Error: %s", e.message)
}

func CreateClient() *ipc.Client {
	config := &ipc.ClientConfig{Encryption: true}
	cc, err := ipc.StartClient("jimssocket", config)
	if err != nil {
		log.Fatal("Could not create ipc client. Reason:", err)
	}

	return cc
}

func IsServerStatusReady(client *ipc.Client) (bool, error) {
	attempt := 0

	writetoServer(client, ReqStatus, []byte("dummy message"))

	for {
		if attempt > 2 {
			return false, &GenericError{"Unable to retrieve server status"}
		}

		m, err := client.Read()

		if err != nil {
			log.Fatal("IPC Communication breakdown, reason: ", err.Error())
		}

		switch m.MsgType {
		case -2: // message type -2 is an error, these won't automatically cause the recieve channel to close.
			log.Println("Error reading from socket: " + err.Error())
			attempt++
		case ResNeedDecryption:
			return false, nil
		case ResReadyToServe:
			return true, nil
		default:
			log.Println("Client received unexpected message: " + fmt.Sprint(m.MsgType) + ": " + string(m.Data))
		}

	}
}

func writetoServer(client *ipc.Client, msgType int, message []byte) {
	for {
		if client.Status() != "Connected" {
			time.Sleep(1 * time.Second)
		}
		err := client.Write(msgType, message)
		if err != nil {
			log.Println("Error writing to server:", err.Error())
			switch err.Error() {
			case "Error":
				log.Fatal("Failed to write to the server. Application will exit, start again.")
			case "Timeout":
				log.Fatal("Reached timeout when connecting to the server, is the daemon up and running?")
			case "Closed":
				log.Fatal("Failed to write to the server. Application will exit, start again.")
			default:
				log.Println("State update: ", err.Error())
			}
		} else {
			break
		}
	}
}

func readMessage(client *ipc.Client) (interface{}, error) {
	errorCounter := 0
	for {
		m, err := client.Read()

		if err != nil {
			log.Fatal("IPC Communication breakdown. Please start again. If the error persists, the daemon process should be killed.")
		}
		switch m.MsgType {
		case -1: // message type -1 is status change and only used internally
			// once a verbosity flag is implemented, this may print additional information
			//log.Println("Status: " + m.Status)

		case -2: // message type -2 is an error, these won't automatically cause the recieve channel to close.
			log.Println("Error: " + err.Error())
			errorCounter++
			if errorCounter > 10 {
				log.Fatal("Exhausted retry budget, application will exit. Please try again.")
			}
		default:
			log.Println("Client received unexpected message: " + fmt.Sprint(m.MsgType) + ": " + string(m.Data))
		}

	}

}
