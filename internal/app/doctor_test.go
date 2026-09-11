package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mangile-cli/internal/cfg"
)

func writeDoctorSeries(t *testing.T, root, name, config string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if config != "" {
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDoctorFindsLocalIssues(t *testing.T) {
	root := t.TempDir()
	writeDoctorSeries(t, root, "Sorunlu", "type: manga\ntitle: Sorunlu\nuploadStatus: hatali\ntags:\n- Uydurma\n")
	writeDoctorSeries(t, root, "BosKlasor", "myAnimeListId: 5\ntype: manga\ntitle: Bos\n")
	if err := os.MkdirAll(filepath.Join(root, "BosKlasor", "Bölüm 1"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := New(cfg.Config{ProjectID: "p", Dataset: "d", APIVersion: "v", UploadsDir: root})
	if err := a.Doctor(context.Background()); err == nil {
		t.Fatal("sorunlu dizin hata vermeli")
	}
}

func TestDoctorClean(t *testing.T) {
	root := t.TempDir()
	writeDoctorSeries(t, root, "Temiz", "myAnimeListId: 7\ntype: manga\ntitle: Temiz\nuploadStatus: uploading\ntags:\n- Aksiyon\n")
	a := New(cfg.Config{ProjectID: "p", Dataset: "d", APIVersion: "v", UploadsDir: root})
	if err := a.Doctor(context.Background()); err != nil {
		t.Fatalf("temiz dizin geçmeli: %v", err)
	}
}

func TestDoctorMissingConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "YamlYok"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := New(cfg.Config{ProjectID: "p", Dataset: "d", APIVersion: "v", UploadsDir: root})
	if err := a.Doctor(context.Background()); err == nil {
		t.Fatal("config.yaml'siz dizin sorun sayılmalı")
	}
}
