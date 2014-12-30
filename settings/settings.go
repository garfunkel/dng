package settings

import (
	"encoding/json"
	"io/ioutil"
)

const (
	SettingsPath = "settings/settings.json"
)

type settings struct {
	Host string `json:"host"`
	Port int `json:"port"`
	Addresses []string `json:"addresses"`
	Landmarks []string `json:"landmarks"`
}

var Settings settings

func ReadSettings() (err error) {
	data, err := ioutil.ReadFile(SettingsPath)

	if err != nil {
		return
	}

	err = json.Unmarshal(data, &Settings)

	return
}
