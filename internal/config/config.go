package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	JavaPath            string `json:"java_path"`
	JarName             string `json:"jar_name"`
	RAMGB               int    `json:"ram_gb"`
	PlayitPath          string `json:"playit_path"`
	BackupRetentionDays int    `json:"backup_retention_days"`
	Port                int    `json:"port"`
}

const configFileName = "config.json"

func DefaultConfig() *Config {
	return &Config{
		JavaPath:            "java",
		JarName:             "server.jar",
		RAMGB:               4,
		PlayitPath:          "playit.exe",
		BackupRetentionDays: 7,
		Port:                25565,
	}
}

func Load() (*Config, error) {
	if _, err := os.Stat(configFileName); os.IsNotExist(err) {
		defaultCfg := DefaultConfig()
		if err := saveConfig(defaultCfg); err != nil {
			return nil, err
		}
		return defaultCfg, nil
	}

	file, err := os.ReadFile(configFileName)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFileName, data, 0644)
}
