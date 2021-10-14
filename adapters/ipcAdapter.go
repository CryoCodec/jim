package adapters

import (
	"encoding/json"
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	"github.com/CryoCodec/jim/core/ports"
	"github.com/CryoCodec/jim/ipc"
	"github.com/pkg/errors"
)

type ipcAdapterImpl struct {
	client *ipc.IpcClient
}

// InstantiateAdapter instantiates an implementation of the IpcPort
func InstantiateAdapter(client *ipc.IpcClient) ports.IpcPort {
	return &ipcAdapterImpl{client: client}
}

// LoadConfigFile causes the server to load the config file.
func (adapter *ipcAdapterImpl) LoadConfigFile(path string) error {
	client := adapter.client
	client.WriteToServer(domain.ReqLoadFile, []byte(path))
	switch response := client.ListenForAnswer(); response.Code {
	case domain.ResError:
		return fmt.Errorf("server failed to load config file, reason: %s", string(response.Payload))
	case domain.ResSuccess:
		return nil
	default:
		return fmt.Errorf("received unexpected message %s, when loading config file", domain.MsgCodeToString[uint16(response.Code)])
	}
}

// GetMatchingServer asks the server for a matching entry for the query string.
// The server has to be in ready state.
func (adapter *ipcAdapterImpl) GetMatchingServer(query string) (*domain.MatchResponse, error) {
	client := adapter.client
	client.WriteToServer(domain.ReqClosestMatch, []byte(query))
	message := client.ListenForAnswer()
	switch message.Code {
	case domain.ResClosestMatch:
		result, err := domain.UnmarshalMatchResponse(message.Payload)
		if err != nil {
			return nil, errors.Errorf("Failed to deserialize json response. This is likely an implementation bug. Reason: %s", err.Error())
		}
		return &result, nil
	case domain.ResNoMatch:
		return nil, errors.Errorf("No Server matched your query.")
	case domain.ResError:
		return nil, errors.Errorf("Server error: %s", string(message.Payload))
	default:
		return nil, errors.Errorf("Received unexpected message %s, when requesting entries.", domain.MsgCodeToString[uint16(message.Code)])
	}
}

// GetEntries asks the server for all entries in the config file and returns these.
// The server has to be in ready state.
func (adapter *ipcAdapterImpl) GetEntries() (chan domain.ListResponseElement, error) {
	out := make(chan domain.ListResponseElement)
	client := adapter.client
	client.WriteToServer(domain.ReqListEntries, []byte{})
	go func() {
		for {
			message := client.ListenForAnswer()
			switch message.Code {
			case domain.ResListEntries:
				result, err := domain.UnmarshalListResponseElement(message.Payload)
				if err != nil {
					panic(fmt.Sprintf("Failed to deserialize json response. This is likely an implementation bug. Reason: %s", err.Error()))
				}
				out <- result
			case domain.ResError:
				panic(fmt.Sprintf("Server error: %s", string(message.Payload)))
			case domain.ResSuccess:
				close(out)
				return
			default:
				panic(fmt.Sprintf("Received unexpected message %s, when requesting entries.", domain.MsgCodeToString[uint16(message.Code)]))
			}
		}
	}()
	return out, nil
}

// MatchClosestN gets a list of potentially matching entries in the config file
func (adapter *ipcAdapterImpl) MatchClosestN(query string) []string {
	for {
		client := adapter.client
		client.WriteToServer(domain.ReqClosestN, []byte(query))
		message := client.ListenForAnswer()
		switch message.Code {
		case domain.ResClosestN:
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

// IsServerReady checks whether the server is ready to serve
func (adapter *ipcAdapterImpl) IsServerReady() bool {
	client := adapter.client
	client.WriteToServer(domain.ReqStatus, []byte{})
	message := client.ListenForAnswer()
	switch message.Code {
	case domain.ResReadyToServe:
		return true
	default:
		return false
	}
}

// ServerStatus queries and returns the server state.
func (adapter *ipcAdapterImpl) ServerStatus() domain.Code {
	client := adapter.client
	client.WriteToServer(domain.ReqStatus, []byte{})
	message := client.ListenForAnswer()
	return message.Code
}

// ServerStatus queries and returns the server state.
func (adapter *ipcAdapterImpl) AttemptDecryption(password []byte) domain.Code {
	client := adapter.client
	client.WriteToServer(domain.ReqAttemptDecryption, password)
	message := client.ListenForAnswer()
	return message.Code
}

// Close closes the underlying ipc connection
func (adapter *ipcAdapterImpl) Close() {
	adapter.client.Close()
}
