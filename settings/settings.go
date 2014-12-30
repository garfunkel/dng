// Package settings allows for configuration settings in dng.
package settings

import (
	"encoding/json"
	"io/ioutil"
)

const (
	// SettingsPath is the path to the settings file.
	SettingsPath = "settings/settings.json"
)

// settings is the settings type, storing configuration options.
type settings struct {
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	DBPath    string   `json:"dbpath"`
	Addresses []string `json:"addresses"`
	Landmarks []string `json:"landmarks"`
}

// Settings is the global settings manager.
var Settings settings

// ReadSettings reads settings from the config file.
func ReadSettings() (err error) {
	data, err := ioutil.ReadFile(SettingsPath)

	if err != nil {
		return
	}

	err = json.Unmarshal(data, &Settings)

	return
}
