package config

import "encoding/json"

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
	Group  string         `json:"group"`
	Env    string         `json:"env"`
	Tag    string         `json:"tag"`
	Server JimConfigEntry `json:"server"`
}

// JimConfigEntry holds all the information necessary to connect to a server via ssh
type JimConfigEntry struct {
	Host     string `json:"host"`
	Dir      string `json:"dir"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}
