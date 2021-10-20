package services

import (
	factory "github.com/CryoCodec/jim/adapters"
	"github.com/CryoCodec/jim/core/domain"
	"github.com/CryoCodec/jim/core/ports"
	"github.com/CryoCodec/jim/files"
	"log"
	"sort"
)

type UiService interface {
	// GetEntries tries to fetch all configured server items.
	// Whenever the server is not yet ready, the error will indicate this.
	GetEntries() (*domain.GroupList, error)

	// GetMatchingServer requests a server entry from the daemon, that matches the given query string.
	// Requires the daemon to be in ready state.
	GetMatchingServer(query string) (*domain.Match, error)

	// MatchClosestN gets a list of n potentially matching entries in the config file.
	// Requires the daemon to be in ready state.
	MatchClosestN(query string) []string

	// Decrypt attempts to decrypt the config file on the server.
	// Before calling this method ensure the server is in the right state
	// to accept a password.
	Decrypt(password []byte) error

	// ReloadConfigFile makes the server reload the config file.
	// This method sets the server to a new state, requiring a password
	// for decryption.
	ReloadConfigFile() error

	// IsServerReady queries the server state. If it has successfully loaded the
	// config file and is decrypted, it is considered ready.
	IsServerReady() bool

	// GetState queries the server state.
	GetState() (*domain.ServerState, error)

	// ShutDown cleans up resources used for server communication.
	ShutDown()
}

type UiServiceImpl struct {
	ipcPort ports.IpcPort
}

func (u *UiServiceImpl) GetEntries() (*domain.GroupList, error) {
	ipcPort := u.ipcPort

	list, err := ipcPort.GetEntries()
	if err != nil {
		return nil, err
	}

	sort.Sort(list)
	for _, group := range *list {
		sort.Sort(group.Entries)
	}
	return list, nil
}

func (u *UiServiceImpl) GetMatchingServer(query string) (*domain.Match, error) {
	return u.ipcPort.GetMatchingServer(query)
}

func (u *UiServiceImpl) MatchClosestN(query string) []string {
	return u.ipcPort.MatchClosestN(query)
}

func (u *UiServiceImpl) Decrypt(password []byte) error {
	ipcPort := u.ipcPort
	err := ipcPort.AttemptDecryption(password)

	if err != nil {
		return err
	}

	return nil
}

func (u *UiServiceImpl) ReloadConfigFile() error {
	path, err := files.GetJimConfigFilePath()
	if err != nil {
		return err
	}

	err = u.ipcPort.LoadConfigFile(path)
	if err != nil {
		log.Printf("UiService: error = %s", err)
		return err
	}

	log.Printf("UiService: got no error, returning nil")
	return nil
}

func (u *UiServiceImpl) IsServerReady() bool {
	return u.ipcPort.IsServerReady()
}

func (u *UiServiceImpl) GetState() (*domain.ServerState, error) {
	state, err := u.ipcPort.ServerStatus()
	if err != nil {
		return nil, err
	}

	return state, nil
}
func (u *UiServiceImpl) ShutDown() {
	u.ipcPort.Close()
}

// NewUiService is the factory method for creating a UiService object.
func NewUiService(verboseLogging bool) UiService {
	// TODO treat the verboseLogging flag properly
	ipcPort := factory.InstantiateAdapter(factory.InitializeGrpcContext())
	return &UiServiceImpl{ipcPort: ipcPort}
}
