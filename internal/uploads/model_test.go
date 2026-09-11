package uploads

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeriesConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := SeriesConfig{
		SanityID:      "abc",
		MalID:         42,
		Type:          "manga",
		Title:         "Test Seri",
		UploadStatus:  "uploading",
		Tags:          []string{"Aksiyon"},
		DefaultScanID: "scan-1",
	}
	if err := SaveSeriesConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeriesConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != cfg.Title || got.MalID != cfg.MalID || got.Type != cfg.Type ||
		got.SanityID != cfg.SanityID || got.UploadStatus != cfg.UploadStatus ||
		len(got.Tags) != 1 || got.Tags[0] != "Aksiyon" || got.DefaultScanID != cfg.DefaultScanID {
		t.Errorf("roundtrip farklı: %+v", got)
	}
}

func TestDiscoverSeriesSkipsMissingConfig(t *testing.T) {
	uploads := t.TempDir()
	if err := os.MkdirAll(filepath.Join(uploads, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	dirB := filepath.Join(uploads, "B")
	if err := SaveSeriesConfig(dirB, SeriesConfig{Title: "B", MalID: 7, Type: "lightNovel"}); err != nil {
		t.Fatal(err)
	}
	series, err := DiscoverSeries(uploads)
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 2 {
		t.Fatalf("2 seri beklenir, %d bulundu", len(series))
	}
	if !series[1].IsNovel() || series[1].Name() != "B" {
		t.Errorf("B serisi hatalı: %+v", series[1])
	}
}

func TestFindSeries(t *testing.T) {
	uploads := t.TempDir()
	_ = SaveSeriesConfig(filepath.Join(uploads, "X"), SeriesConfig{Title: "Seri Adı"})
	s, err := FindSeries(uploads, "Seri Adı")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(s.Dir) != "X" {
		t.Errorf("beklenen X, %s", filepath.Base(s.Dir))
	}
	if _, err := FindSeries(uploads, "Yok"); err == nil {
		t.Fatal("bulunamayan seride hata beklenir")
	}
}
