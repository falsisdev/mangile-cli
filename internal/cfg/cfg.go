package cfg

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mangile-cli/internal/constants"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ProjectID  string
	Dataset    string
	APIVersion string
	Token      string
	UploadsDir string
	WebPort    int
	Version    string
	DryRun     bool
}

type UserConfig struct {
	UploadsPath string `yaml:"uploadsPath"`
}

func Load() Config {
	cfg := Config{
		ProjectID:  envOr("MANGILE_PROJECT_ID", constants.ProjectIDDefault),
		Dataset:    envOr("MANGILE_DATASET", constants.DatasetDefault),
		APIVersion: envOr("MANGILE_API_VERSION", constants.APIVersionDefault),
		Token:      os.Getenv("SANITY_TOKEN"),
	}
	cfg.UploadsDir = resolveUploadsDir()
	cfg.WebPort = resolveWebPort()
	return cfg
}

func (c Config) TokenOrEmpty() string {
	return c.Token
}

func (c Config) HasToken() bool {
	return c.Token != ""
}

func (c Config) UploadsPath() string {
	return filepath.Clean(c.UploadsDir)
}

func resolveUploadsDir() string {
	if v := os.Getenv("MANGILE_UPLOADS"); v != "" {
		return v
	}
	if uc, err := LoadUserConfig(); err == nil && uc.UploadsPath != "" {
		return uc.UploadsPath
	}
	return "uploads"
}

var configDirOverride = ""

func UserConfigDir() (string, error) {
	dir := configDirOverride
	if dir == "" {
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(dir, "mangile"), nil
}

func UserConfigPath() (string, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func LoadUserConfig() (UserConfig, error) {
	var uc UserConfig
	path, err := UserConfigPath()
	if err != nil {
		return uc, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return uc, err
	}
	if err := yaml.Unmarshal(data, &uc); err != nil {
		return uc, err
	}
	return uc, nil
}

func SaveUserConfig(uc UserConfig) error {
	dir, err := UserConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(uc)
	if err != nil {
		return err
	}
	path, err := UserConfigPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func HasUserConfig() bool {
	_, err := os.Stat(mustUserConfigPath())
	return err == nil
}

func mustUserConfigPath() string {
	path, err := UserConfigPath()
	if err != nil {
		return ""
	}
	return path
}

func DefaultUploadsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "mangile"
	}
	return filepath.Join(home, "Documents", "mangile")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func resolveWebPort() int {
	raw := strings.TrimSpace(os.Getenv("MANGILE_WEB_PORT"))
	if raw == "" {
		return constants.WebServerPort
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return constants.WebServerPort
	}
	return port
}
