package services

import (
	factory "github.com/CryoCodec/jim/adapters"
	"github.com/CryoCodec/jim/core/domain"
	"github.com/CryoCodec/jim/core/ports"
	"github.com/CryoCodec/jim/files"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sort"
	"strings"
)

type UiService interface {
	// GetEntries tries to fetch all configured server items.
	// Whenever the server is not yet ready, the error will indicate this.
	GetEntries(filters []string, limit int) (*domain.GroupList, error)

	// GetMatchingServer requests a server entry from the daemon, that matches the given query string.
	// Requires the daemon to be in ready state.
	GetMatchingServer(query string) (*domain.Match, error)

	// MatchClosestN gets a list of n potentially matching entries in the config file.
	// Requires the daemon to be in ready state.
	MatchClosestN(query string) []string

	// Decrypt attempts to decrypt the config file on the server.
	// Before calling this method ensure the server is in the right state
	// to accept a password.
	Decrypt(password []byte) (chan domain.DecryptStep, error)

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

func (u *UiServiceImpl) GetEntries(filters []string, limit int) (*domain.GroupList, error) {
	filter, err := parseFilters(filters)
	if err != nil {
		return nil, err
	}

	ipcPort := u.ipcPort

	list, err := ipcPort.GetEntries(filter, limit)
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

func (u *UiServiceImpl) Decrypt(password []byte) (chan domain.DecryptStep, error) {
	ipcPort := u.ipcPort
	channel, err := ipcPort.AttemptDecryption(password)

	if err != nil {
		return nil, err
	}

	return channel, nil
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
func NewUiService() UiService {
	ipcPort := factory.InstantiateAdapter(factory.InitializeGrpcContext())
	return &UiServiceImpl{ipcPort: ipcPort}
}

func parseFilters(filters []string) (*domain.Filter, error) {
	filter := &domain.Filter{}
	for _, filterString := range filters {
		if filterString == "" {
			continue
		}

		slice := strings.SplitN(filterString, ":", 2)
		if len(slice) == 1 { // no prefix used, so it's the free filter.
			log.Tracef("Setting free filter: %s", slice[0])
			filter.FreeFilter = slice[0]
			continue
		}

		// from here on every filter only applies to a given category
		if len(slice) != 2 {
			return nil, errors.Errorf("Encountered invalid filter: %s", filterString)
		}

		log.Tracef("Setting filter with category %s and value %s", slice[0], slice[1])
		switch strings.ToLower(slice[0]) {
		case "env":
			filter.EnvFilter = slice[1]
		case "tag":
			filter.TagFilter = slice[1]
		case "group":
			filter.GroupFilter = slice[1]
		case "host":
			filter.HostFilter = slice[1]
		default:
			return nil, errors.Errorf("Encountered invalid filter category: %s in %s", slice[1], filterString)
		}
	}
	return filter, nil
}
