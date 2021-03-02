package model

import "encoding/json"

type JimConfig []JimConfigElement

func UnmarshalJimConfig(data []byte) (JimConfig, error) {
	var r JimConfig
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *JimConfig) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type JimConfigElement struct {
	Env    string `json:"env"`
	Tag    string `json:"tag"`
	Server Server `json:"server"`
}

type Server struct {
	Host     string `json:"host"`
	Dir      string `json:"dir"`
	Username string `json:"username"`
	Password string `json:"password"`
}
