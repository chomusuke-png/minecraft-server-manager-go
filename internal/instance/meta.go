package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type InstanceMeta struct {
	LoaderType    string `json:"loader_type"` // "paper", "fabric", "forge", "vanilla"
	MCVersion     string `json:"mc_version"`
	LoaderVersion string `json:"loader_version,omitempty"` // build de Paper, versión de loader de Fabric/Forge; vacío en Vanilla
	RAMGB         int    `json:"ram_gb,omitempty"`         // 0 = usar el valor global de config.json
	// LaunchArgs reemplaza al '-jar server.jar' del arranque normal en loaders que
	// no producen un jar ejecutable (Forge >= 1.17). Vacío en el resto.
	LaunchArgs []string `json:"launch_args,omitempty"`
	// JavaPath pisa el java_path global. Hace falta porque cada versión de
	// Minecraft exige un major distinto y no pueden convivir en uno solo.
	JavaPath string `json:"java_path,omitempty"`
	// BackupKeepMin pisa el backup_keep_min global. 0 = usar el valor global.
	BackupKeepMin int `json:"backup_keep_min,omitempty"`
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
