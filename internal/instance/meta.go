package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type InstanceMeta struct {
	LoaderType    string `json:"loader_type"` // "paper", "fabric", "vanilla"
	MCVersion     string `json:"mc_version"`
	LoaderVersion string `json:"loader_version,omitempty"` // build de Paper, versión de loader de Fabric; vacío en Vanilla
	RAMGB         int    `json:"ram_gb,omitempty"`         // 0 = usar el valor global de config.json
	Port          int    `json:"port,omitempty"`
}

func SaveMeta(instanceDir string, meta InstanceMeta) error {
	path := filepath.Join(instanceDir, "instance.json")
	data, err := json.MarshalIndent(meta, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadMeta(instanceDir string) (*InstanceMeta, error) {
	path := filepath.Join(instanceDir, "instance.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no se encontró metadata de la instancia: %w", err)
	}
	var meta InstanceMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}
