package domain

import (
	"github.com/pkg/errors"
)

// Match is used in the client server communication as the reponse format of the connect command
type Match struct {
	Tag    string
	Server Server
}

// Server holds all the information necessary to connect to a server via ssh
type Server struct {
	Host     string
	Dir      string
	Port     int
	Username string
	Password []byte
}

type GroupList []Group

type Group struct {
	Title   string
	Entries ConnectionList
}

func (a GroupList) Len() int           { return len(a) }
func (a GroupList) Less(i, j int) bool { return a[i].Title < a[j].Title }
func (a GroupList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// ConnectionList is a type alias for a list of ListResponseElements
type ConnectionList []ConnectionInfo

// functions used to implement the sort interface
func (a ConnectionList) Len() int           { return len(a) }
func (a ConnectionList) Less(i, j int) bool { return a[i].Tag < a[j].Tag }
func (a ConnectionList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// ConnectionInfo is used in the client server communication as a response format in the list command
type ConnectionInfo struct {
	Tag      string
	HostInfo string
}

const (
	RequiresConfigFile = iota
	RequiresDecryption
	Ready
)

type ServerState struct {
	state int
}

func (s *ServerState) IsReady() bool {
	return s.state == Ready
}

func (s *ServerState) RequiresConfigFile() bool {
	return s.state == RequiresConfigFile
}

func (s *ServerState) RequiresDecryption() bool {
	return s.state == RequiresDecryption
}

func NewServerState(state int) (*ServerState, error) {
	switch state {
	case RequiresConfigFile:
		return &ServerState{state: state}, nil
	case RequiresDecryption:
		return &ServerState{state: state}, nil
	case Ready:
		return &ServerState{state: state}, nil
	default:
		return nil, errors.Errorf("Unknown server state as parameter: %d", state)
	}
}
