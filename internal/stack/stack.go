package stack

import (
	"encoding/json"
	"fmt"
	"os"
)

type Catalog map[string]Config

type Config struct {
	AppPort  int      `json:"app_port"`
	Commands Commands `json:"commands"`
	Docker   Docker   `json:"docker"`
}

type Commands struct {
	Install         string `json:"install"`
	Lint            string `json:"lint"`
	Type            string `json:"type"`
	Test            string `json:"test"`
	TestIntegration string `json:"test_integration"`
	MigrateCheck    string `json:"migrate_check"`
	Build           string `json:"build"`
	Scan            string `json:"scan"`
	Start           string `json:"start"`
}

type Docker struct {
	BaseImage   string `json:"base_image"`
	CopyDeps    string `json:"copy_deps"`
	InstallStep string `json:"install_step"`
	BuildStep   string `json:"build_step"`
	StartCmd    string `json:"start_cmd"`
}

func LoadCatalog(path string) (Catalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read stack config: %w", err)
	}
	var c Catalog
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse stack config: %w", err)
	}
	return c, nil
}

func (c Catalog) Get(name string) (Config, error) {
	stack, ok := c[name]
	if !ok {
		return Config{}, fmt.Errorf("unknown stack: %s", name)
	}
	if err := validate(stack); err != nil {
		return Config{}, err
	}
	return stack, nil
}

func validate(cfg Config) error {
	if cfg.AppPort == 0 {
		return fmt.Errorf("missing app_port")
	}
	if cfg.Commands.Install == "" || cfg.Commands.Lint == "" || cfg.Commands.Type == "" || cfg.Commands.Test == "" || cfg.Commands.TestIntegration == "" || cfg.Commands.MigrateCheck == "" || cfg.Commands.Build == "" || cfg.Commands.Scan == "" || cfg.Commands.Start == "" {
		return fmt.Errorf("missing required command values")
	}
	if cfg.Docker.BaseImage == "" || cfg.Docker.CopyDeps == "" || cfg.Docker.InstallStep == "" || cfg.Docker.BuildStep == "" || cfg.Docker.StartCmd == "" {
		return fmt.Errorf("missing docker config values")
	}
	return nil
}
