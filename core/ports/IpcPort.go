package ports

import (
	"github.com/CryoCodec/jim/core/domain"
)

// IpcPort defines the port for the interprocess communication with the jim daemon.
type IpcPort interface {
	// LoadConfigFile requests the daemon process to load a config file
	LoadConfigFile(path string) error
	// AttemptDecryption requests a decryption attempt from the daemon, using the passed password
	AttemptDecryption(password []byte) (chan domain.DecryptStep, error)
	// GetMatchingServer requests a server entry from the daemon, that matches the given query string.
	// Requires the daemon to be in ready state.
	GetMatchingServer(query string) (*domain.Match, error)
	// GetEntries requests all entries of the loaded config from the daemon.
	// Requires the daemon to be in ready state.
	GetEntries(filter *domain.Filter, limit int) (*domain.GroupList, error)
	// MatchClosestN gets a list of n potentially matching entries in the config file.
	// Requires the daemon to be in ready state.
	MatchClosestN(query string) []string
	// IsServerReady queries the server state. The server is in ready state,
	// if a config file was loaded successfully and decrypted.
	IsServerReady() bool
	// ServerStatus queries and returns the server state.
	ServerStatus() (*domain.ServerState, error)
	// Close closes the underlying ipc connection
	Close() error
}
