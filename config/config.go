package config

import (
	"encoding/json"
	"os"
)

type DSNConfig struct {
	Driver            string `json:"driver"`
	ExportDatabaseDSN string `json:"export_database_dsn"`
	ImportDatabaseDSN string `json:"import_database_dsn"`
}

// LoadDSNConfig loads DSNConfig from the specified JSON file.
func LoadDSNConfig(filename string) (*DSNConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config DSNConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}
