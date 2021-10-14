package services

import (
	factory "github.com/CryoCodec/jim/adapters"
	"github.com/CryoCodec/jim/core/domain"
	"github.com/CryoCodec/jim/core/ports"
	"github.com/CryoCodec/jim/files"
	"github.com/CryoCodec/jim/ipc"
	"github.com/pkg/errors"
	"log"
	"sort"
)

type UiService interface {
	// GetEntries tries to fetch all configured server items.
	// Whenever the server is not yet ready, the error will indicate this.
	GetEntries() (domain.ListEntries, error)

	// GetMatchingServer requests a server entry from the daemon, that matches the given query string.
	// Requires the daemon to be in ready state.
	GetMatchingServer(query string) (*domain.MatchResponse, error)

	// MatchClosestN gets a list of n potentially matching entries in the config file.
	// Requires the daemon to be in ready state.
	MatchClosestN(query string) []string

	// Decrypt attempts to decrypt the config file on the server.
	// If the decryption fails due to wrong password the bool is set to false.
	// In case of a serious error, the error is set to a value.
	Decrypt(password []byte) (bool, error)

	// ReloadConfigFile makes the server reload the config file
	ReloadConfigFile() error

	// IsServerReady queries the server state. If it has successfully loaded the
	// config file and is decrypted, it is considered ready.
	IsServerReady() bool

	// ShutDown cleans up resources used for server communication.
	ShutDown()
}

type UiServiceImpl struct {
	ipcPort ports.IpcPort
}

func (u *UiServiceImpl) GetEntries() (domain.ListEntries, error) {
	ipcPort := u.ipcPort
	err := runPreamble(ipcPort)

	if err != nil {
		return nil, err
	}

	var responseElements domain.ListResponse
	c, err := ipcPort.GetEntries()
	if err != nil {
		return nil, err
	}

	for el := range c {
		responseElements = append(responseElements, el)
	}

	var result domain.ListEntries
	for _, el := range responseElements {
		mapped := domain.ListEntry{
			Title:   el.Title,
			Content: el.Content,
		}
		result = append(result, mapped)
	}

	sort.Sort(result)
	return result, nil
}

func (u *UiServiceImpl) GetMatchingServer(query string) (*domain.MatchResponse, error) {
	return u.ipcPort.GetMatchingServer(query)
}

func (u *UiServiceImpl) MatchClosestN(query string) []string {
	return u.ipcPort.MatchClosestN(query)
}

func (u *UiServiceImpl) Decrypt(password []byte) (bool, error) {
	ipcPort := u.ipcPort
	err := runPreamble(ipcPort)

	if err != nil {
		serviceError, ok := err.(domain.ServiceError)
		if ok && !serviceError.IsPasswordRequired() {
			return false, err
		} else {
			return false, err
		}
	}

	response := ipcPort.AttemptDecryption(password)

	switch response {
	case domain.ResDecryptionFailed:
		return false, nil
	case domain.ResError:
		return false, errors.Errorf("received unexpected error from server.")
	default:
		return false, errors.Errorf("Got unexpected response from server %s", domain.MsgCodeToString[uint16(response)])
	}
}

func (u *UiServiceImpl) ReloadConfigFile() error {
	path, err := files.GetJimConfigFilePath()
	if err != nil {
		return err
	}

	err = u.ipcPort.LoadConfigFile(path)
	if err != nil {
		return err
	}

	return nil
}

func (u *UiServiceImpl) IsServerReady() bool {
	return u.ipcPort.IsServerReady()
}

func (u *UiServiceImpl) ShutDown() {
	u.ipcPort.Close()
}

// NewUiService is the factory method for creating a UiService object.
func NewUiService(verboseLogging bool) UiService {
	ipcPort := factory.InstantiateAdapter(ipc.InitializeClient(verboseLogging))
	return &UiServiceImpl{ipcPort: ipcPort}
}

func runPreamble(ipcPort ports.IpcPort) error {
	for {
		serverState := ipcPort.ServerStatus()
		switch serverState {
		case domain.ResReadyToServe:
			return nil
		case domain.ResRequireConfigFile:
			if err := loadConfigFile(ipcPort); err != nil {
				return err
			}
		case domain.ResNeedDecryption:
			return domain.NewServiceError(domain.RequiresDecryption)
		default:
			return errors.Errorf("Received unexpected server response in preamble: %s", domain.MsgCodeToString[uint16(serverState)])
		}

	}
}

func loadConfigFile(ipcPort ports.IpcPort) error {
	path, err := files.GetJimConfigFilePath()
	if err != nil {
		return err
	}
	log.Printf("Trying to load config file from %s", path)
	return ipcPort.LoadConfigFile(path)
}
