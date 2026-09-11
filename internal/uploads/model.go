package uploads

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type SeriesConfig struct {
	SanityID      string   `yaml:"sanityId"`
	MalID         int      `yaml:"myAnimeListId"`
	Type          string   `yaml:"type"`
	Title         string   `yaml:"title"`
	Slug          string   `yaml:"slug"`
	Format        string   `yaml:"format"`
	UploadStatus  string   `yaml:"uploadStatus"`
	Tags          []string `yaml:"tags"`
	Description   string   `yaml:"description"`
	CoverImage    string   `yaml:"coverImage"`
	BannerImage   string   `yaml:"bannerImage"`
	DefaultScanID string   `yaml:"defaultScanId"`
}

type Series struct {
	Dir    string
	Config SeriesConfig
}

func (s *Series) Name() string {
	if s.Config.Title != "" {
		return s.Config.Title
	}
	return filepath.Base(s.Dir)
}

func (s *Series) IsManga() bool {
	return s.Config.Type == "" || s.Config.Type == "manga"
}

func (s *Series) IsNovel() bool {
	return s.Config.Type == "lightNovel"
}

func (s *Series) IsConfigured() bool {
	return s.Config.SanityID != "" || s.Config.MalID != 0
}

func LoadSeriesConfig(dir string) (SeriesConfig, error) {
	var cfg SeriesConfig
	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func SaveSeriesConfig(dir string, cfg SeriesConfig) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0o644)
}

func WriteTemplateConfig(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err == nil {
		return nil
	}
	return SaveSeriesConfig(dir, SeriesConfig{UploadStatus: "uploading"})
}

func DiscoverSeries(uploadsDir string) ([]Series, error) {
	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var series []Series
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dir := filepath.Join(uploadsDir, e.Name())
		cfg, err := LoadSeriesConfig(dir)
		if err != nil {
			cfg = SeriesConfig{Type: "manga"}
		}
		series = append(series, Series{Dir: dir, Config: cfg})
	}
	return series, nil
}

func FindSeries(uploadsDir, name string) (Series, error) {
	all, err := DiscoverSeries(uploadsDir)
	if err != nil {
		return Series{}, err
	}
	for _, s := range all {
		if s.Name() == name || filepath.Base(s.Dir) == name {
			return s, nil
		}
	}
	return Series{}, fmt.Errorf("seri bulunamadı: %s", name)
}
