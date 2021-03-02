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
	Group  string `json:"group"`
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

type ListResponse []ListResponseElement

func UnmarshalListResponse(data []byte) (ListResponse, error) {
	var r ListResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ListResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type ListResponseElement struct {
	Title   string   `json:"title"`
	Content []string `json:"content"`
}

func (a ListResponse) Len() int           { return len(a) }
func (a ListResponse) Less(i, j int) bool { return a[i].Title < a[j].Title }
func (a ListResponse) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
