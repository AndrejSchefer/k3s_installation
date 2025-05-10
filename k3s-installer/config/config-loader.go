package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadConfig reads the JSON config file and unmarshals it into AppConfig
func LoadConfig(filename string) (*AppConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var config AppConfig
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("could not decode JSON: %v", err)
	}

	return &config, nil
}
