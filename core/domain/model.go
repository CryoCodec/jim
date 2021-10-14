package domain

import (
	"encoding/json"
	"fmt"
)

// JimConfig is a type alias for a list of config elements
type JimConfig []JimConfigElement

// UnmarshalJimConfig tries to parse given byte[] in json format to a JimConfig struct
func UnmarshalJimConfig(data []byte) (JimConfig, error) {
	var r JimConfig
	err := json.Unmarshal(data, &r)
	return r, err
}

// Marshal serializes a JimConfig struct to json format
func (r *JimConfig) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// JimConfigElement is the main structure in the json format
type JimConfigElement struct {
	Group  string `json:"group"`
	Env    string `json:"env"`
	Tag    string `json:"tag"`
	Server Server `json:"server"`
}

// Server holds all the information necessary to connect to a server via ssh
type Server struct {
	Host     string `json:"host"`
	Dir      string `json:"dir"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// MatchResponse is used in the client server communication as the reponse format of the connect command
type MatchResponse struct {
	Connection string `json:"connection"`
	Server     Server `json:"server"`
}

// UnmarshalMatchResponse parses the given byte[] in json format into a MatchResponse struct
func UnmarshalMatchResponse(data []byte) (MatchResponse, error) {
	var s MatchResponse
	err := json.Unmarshal(data, &s)
	return s, err
}

// Marshal serializes a MatchResponse struct to json
func (s *MatchResponse) Marshal() ([]byte, error) {
	return json.Marshal(s)
}

// ListResponse is a type alias for a list of ListResponseElements
type ListResponse []ListResponseElement

// ListResponseElement is used in the client server communication as a response format in the list command
type ListResponseElement struct {
	Title   string   `json:"title"`
	Content []string `json:"content"`
}

// UnmarshalListResponseElement deserializes given byte[] in json format to a ListReponseElement struct
func UnmarshalListResponseElement(data []byte) (ListResponseElement, error) {
	var r ListResponseElement
	err := json.Unmarshal(data, &r)
	return r, err
}

// Marshal deserializes a ListReponseElement to json
func (r *ListResponseElement) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// ListEntries is used to serve the UI. It contains information about the configured servers.
type ListEntries []ListEntry
type ListEntry struct {
	Title   string
	Content []string
}

// functions used to implement the sort interface
func (a ListEntries) Len() int           { return len(a) }
func (a ListEntries) Less(i, j int) bool { return a[i].Title < a[j].Title }
func (a ListEntries) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type ServiceError struct {
	errorType ServiceCode
}

func (e ServiceError) Error() string {
	return fmt.Sprintf("The server returned: %s.", ServiceCodeToString[e.errorType])
}

func (e *ServiceError) IsPasswordRequired() bool {
	return e.errorType == RequiresDecryption
}

// NewServiceError is used as a constructor to create service errors with a given message code.
func NewServiceError(code ServiceCode) *ServiceError {
	return &ServiceError{code}
}

const (
	RequiresDecryption = iota + 1
)

type ServiceCode int

var ServiceCodeToString = map[ServiceCode]string{
	RequiresDecryption: `decryption required`,
}
