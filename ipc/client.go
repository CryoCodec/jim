package ipc

import (
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	ipc "github.com/james-barrow/golang-ipc"
	"github.com/pkg/errors"
	"log"
	"time"
)

type IpcClient struct {
	cc              *ipc.Client
	propagationChan chan domain.Message
}

// InitializeClient creates an ipc client, which may be used to
// write and receive data from a unix domain socket/named pipe
func InitializeClient(logVerbose bool) *IpcClient {
	config := &ipc.ClientConfig{
		Timeout: 2,
	}
	cc, err := ipc.StartClient("jimssocket", config)
	if err != nil {
		log.Fatal("Could not create ipc client. Reason:", err)
	}
	msgChannel := startReceiving(cc, logVerbose)
	return &IpcClient{cc, msgChannel}
}

// Close closes the ipc connection.
func (ipcClient *IpcClient) Close() {
	libClient := ipcClient.cc
	if libClient != nil {
		libClient.Close()
	}
}

// startReceiving will read from the socket in go routine until forever. Domain specific messages are forwarded via the returned channel.
func startReceiving(client *ipc.Client, verbose bool) chan domain.Message {
	propagationChan := make(chan domain.Message)

	go func() {
		errorCounter := 0
		for {
			m, err := client.Read()

			if err != nil {
				if !(err.Error() == "Client has closed the connection") { // this message will always be sent, once we close the client intentionally
					propagationChan <- domain.Message{Code: domain.ResError, Payload: []byte(fmt.Sprintf("IPC Communication breakdown. Reason: %s ", err.Error()))}
					close(propagationChan)
				}
				return
			}
			switch m.MsgType {
			case -1: // message type -1 is status change and only used internally
				if verbose {
					log.Println("Status update: " + m.Status)
				}
			case -2: // message type -2 is an error, these won't automatically cause the receive channel to close.
				log.Println("Error: " + err.Error())
				errorCounter++
				if errorCounter > 10 {
					propagationChan <- domain.Message{Code: domain.ResError, Payload: []byte("IPC Communication breakdown. Please restart the application.")}
					close(propagationChan)
				}
				time.Sleep(200 * time.Millisecond)
			default:
				if verbose {
					log.Println("Client received message: " + domain.MsgCodeToString[uint16(m.MsgType)])
				}
				propagationChan <- domain.Message{domain.Code(m.MsgType), m.Data}
			}
		}
	}()

	return propagationChan
}

// WriteToServer Sends the given message to the server.
// returns nil if everything went fine, else an error string.
func (ipcClient *IpcClient) WriteToServer(msgType int, message []byte) error {
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
		return errors.Errorf("Error writing to server: %s", err.Error())
	}
	return nil
}

func (ipcClient *IpcClient) ListenForAnswer() domain.Message {
	return <-ipcClient.propagationChan
}
