package app

import (
	"encoding/json"
)

type Metadata map[string]string

type App struct {
	Env      string   `json:"env"`
	Name     string   `json:"name"`
	Addr     string   `json:"addr"`
	Port     int      `json:"port"`
	Metadata Metadata `json:"metadata"`
}

func (m Metadata) ToMap() map[string]string {
	return map[string]string(m)
}

func (a *App) Encode() string {
	byts, _ := json.Marshal(a)
	return string(byts)
}

func (a *App) Decode(byts []byte) {
	json.Unmarshal(byts, a)
}
