package cfg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserConfigRoundTrip(t *testing.T) {
	configDirOverride = t.TempDir()
	t.Cleanup(func() { configDirOverride = "" })

	uc := UserConfig{UploadsPath: filepath.Join(os.TempDir(), "mangile-data")}
	if err := SaveUserConfig(uc); err != nil {
		t.Fatal(err)
	}
	got, err := LoadUserConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.UploadsPath != uc.UploadsPath {
		t.Errorf("roundtrip farklı: %q vs %q", got.UploadsPath, uc.UploadsPath)
	}
	if !HasUserConfig() {
		t.Error("HasUserConfig true olmalı")
	}
}

func TestUserConfigDir(t *testing.T) {
	configDirOverride = t.TempDir()
	t.Cleanup(func() { configDirOverride = "" })

	dir, err := UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "mangile" {
		t.Errorf("beklenen mangile alt klasörü, %q", filepath.Base(dir))
	}
}

func TestResolveUploadsDirNoConfig(t *testing.T) {
	t.Setenv("MANGILE_UPLOADS", "")
	configDirOverride = t.TempDir()
	t.Cleanup(func() { configDirOverride = "" })

	if got := resolveUploadsDir(); got != "uploads" {
		t.Errorf("config ve env yoksa 'uploads' beklenir, %q", got)
	}
}

func TestResolveUploadsDirFromConfig(t *testing.T) {
	t.Setenv("MANGILE_UPLOADS", "")
	configDirOverride = t.TempDir()
	t.Cleanup(func() { configDirOverride = "" })

	want := filepath.Join(os.TempDir(), "cfg-data")
	if err := SaveUserConfig(UserConfig{UploadsPath: want}); err != nil {
		t.Fatal(err)
	}
	if got := resolveUploadsDir(); got != want {
		t.Errorf("config yolundan %q beklenir, %q", want, got)
	}
}

func TestResolveUploadsDirEnvWins(t *testing.T) {
	t.Setenv("MANGILE_UPLOADS", "/env/uploads")
	configDirOverride = t.TempDir()
	t.Cleanup(func() { configDirOverride = "" })

	if err := SaveUserConfig(UserConfig{UploadsPath: "/config/uploads"}); err != nil {
		t.Fatal(err)
	}
	if got := resolveUploadsDir(); got != "/env/uploads" {
		t.Errorf("env öncelikli olmalı, %q", got)
	}
}

func TestDefaultUploadsPath(t *testing.T) {
	p := DefaultUploadsPath()
	if p == "" {
		t.Fatal("varsayılan yol boş olmamalı")
	}
	if filepath.Base(p) != "mangile" {
		t.Errorf("varsayılan dizin mangile olmalı, %q", filepath.Base(p))
	}
}
