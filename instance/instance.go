package instance

import (
	"encoding/json"
)

type Metadata map[string]string

type Instance struct {
	Env      string   `json:"env"`
	App      string   `json:"app"`
	Addr     string   `json:"addr"`
	Port     int      `json:"port"`
	Metadata Metadata `json:"metadata"`
}

func (m Metadata) ToMap() map[string]string {
	return map[string]string(m)
}

func (inst *Instance) Encode() string {
	byts, _ := json.Marshal(inst)
	return string(byts)
}

func (inst *Instance) Decode(byts []byte) {
	json.Unmarshal(byts, inst)
}
