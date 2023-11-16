package config

import (
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// CmdConfig is the configuration for the command line
type CmdConfig struct {
	// manager default http://127.0.0.1:9100
	ManagerAddress string `json:"managerAddress"`
	Username       string `json:"username"`
	AuthKey        []byte `json:"authKey"`
}

func WriteToFile(filename string, cfg *CmdConfig) error {
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(filename, content, 0600)
}

func GetFromFile(filename string) (*CmdConfig, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var cfg CmdConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
