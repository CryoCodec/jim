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

type Filter struct {
	EnvFilter   string
	GroupFilter string
	TagFilter   string
	HostFilter  string
	FreeFilter  string
}

func NewFilter(envFilter string, groupFilter string, tagFilter string, hostFilter string, freeFilter string) Filter {
	return Filter{
		EnvFilter:   envFilter,
		GroupFilter: groupFilter,
		TagFilter:   tagFilter,
		HostFilter:  hostFilter,
		FreeFilter:  freeFilter,
	}
}

func (f Filter) HasEnvFilter() bool {
	return f.EnvFilter != ""
}

func (f Filter) HasGroupFilter() bool {
	return f.GroupFilter != ""
}

func (f Filter) HasTagFilter() bool {
	return f.TagFilter != ""
}

func (f Filter) HasHostFilter() bool {
	return f.HostFilter != ""
}

func (f Filter) HasFreeFilter() bool {
	return f.FreeFilter != ""
}

func (f Filter) IsAnyFilterSet() bool {
	return f.HasEnvFilter() || f.HasTagFilter() || f.HasGroupFilter() || f.HasHostFilter() || f.HasFreeFilter()
}

type Step = int

const (
	Decrypt = iota
	DecodeBase64
	Unmarshal
	Validate
	BuildIndex
	Done
)

// DecryptStep holds updates given by the server
// during decryption
type DecryptStep struct {
	// whether the step was successful or not
	IsSuccess bool
	// what step was performed
	StepType Step
	// the reason for failure, if the step was unsuccessful
	Reason string
	// only used if something went wrong on the protocol side
	Error error
}

func NewSuccessfulDecryptStep(stepType Step) *DecryptStep {
	return &DecryptStep{IsSuccess: true, StepType: stepType}
}

func NewFailedDecryptStep(stepType Step, reason string) *DecryptStep {
	return &DecryptStep{IsSuccess: false, StepType: stepType, Reason: reason}
}

func NewErrorDecryptStep(err error) *DecryptStep {
	return &DecryptStep{Error: err}
}
