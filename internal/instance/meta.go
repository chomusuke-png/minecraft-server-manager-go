package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type InstanceMeta struct {
	LoaderType     string   `json:"loader_type"`               // "paper", "fabric", "forge", "neoforge", "vanilla"
	MCVersion      string   `json:"mc_version"`                //
	LoaderVersion  string   `json:"loader_version,omitempty"`  // build de Paper, versión de loader de Fabric/Forge/NeoForge; vacío en Vanilla
	RAMGB          int      `json:"ram_gb,omitempty"`          // 0 = usar el valor global de config.json
	LaunchArgs     []string `json:"launch_args,omitempty"`     // reemplaza el ejecutable de arranque si el loader no produce su .jar
	JavaPath       string   `json:"java_path,omitempty"`       // pisa el path de java por si la instalacion requiere otra version
	BackupKeepMin  int      `json:"backup_keep_min,omitempty"` // BackupKeepMin pisa el backup_keep_min global. 0 = usar el valor global.
	TunnelProvider string   `json:"tunnel_provider,omitempty"` // "playit", "ngrok", "none".
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
