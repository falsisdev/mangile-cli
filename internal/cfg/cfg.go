package cfg

import (
	"os"
	"path/filepath"

	"mangile-cli/internal/constants"
)

type Config struct {
	ProjectID  string
	Dataset    string
	APIVersion string
	Token      string
	UploadsDir string
	DryRun     bool
}

func Load() Config {
	return Config{
		ProjectID:  envOr("MANGILE_PROJECT_ID", constants.ProjectIDDefault),
		Dataset:    envOr("MANGILE_DATASET", constants.DatasetDefault),
		APIVersion: envOr("MANGILE_API_VERSION", constants.APIVersionDefault),
		Token:      os.Getenv("SANITY_TOKEN"),
		UploadsDir: envOr("MANGILE_UPLOADS", "uploads"),
	}
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
