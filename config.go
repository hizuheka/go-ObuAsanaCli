package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PersonalAccessToken string `yaml:"personal_access_token"`
	WorkspaceID         string `yaml:"workspace_id"`
	// 追加: 複数プロジェクトの管理とデフォルト設定
	Projects       map[string]string `yaml:"projects"`
	DefaultProject string            `yaml:"default_project"`
	Assignees      map[string]string `yaml:"assignees"`
}

type ConfigStore interface {
	Exists() bool
	Load() (*Config, error)
	CreateTemplate() error
}

type yamlConfigStore struct {
	path   string
	appDir string
}

func NewYamlConfigStore() (ConfigStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}
	appDir := filepath.Join(configDir, "asana-cli")
	return &yamlConfigStore{
		path:   filepath.Join(appDir, "config.yaml"),
		appDir: appDir,
	}, nil
}

func (s *yamlConfigStore) Exists() bool {
	_, err := os.Stat(s.path)
	return !os.IsNotExist(err)
}

func (s *yamlConfigStore) Load() (*Config, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// マップがnilの場合の初期化
	if cfg.Assignees == nil {
		cfg.Assignees = make(map[string]string)
	}
	if cfg.Projects == nil {
		cfg.Projects = make(map[string]string)
	}
	return &cfg, nil
}

func (s *yamlConfigStore) CreateTemplate() error {
	if err := os.MkdirAll(s.appDir, 0755); err != nil {
		return err
	}
	template := Config{
		Assignees: map[string]string{"me": "YOUR_GID"},
		Projects: map[string]string{
			"dev":       "PROJECT_GID_1",
			"marketing": "PROJECT_GID_2",
		},
		DefaultProject: "dev",
	}
	data, _ := yaml.Marshal(&template)
	content := []byte("# Asana Config\n# personal_access_token: https://app.asana.com/0/developer-console から取得\n" + string(data))
	return os.WriteFile(s.path, content, 0644)
}
