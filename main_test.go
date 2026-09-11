package main

import (
	"os"
	"path/filepath"
	"testing"
)

func withArgs(t *testing.T, args []string) {
	t.Helper()
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = args
}

func withUploads(t *testing.T) {
	t.Helper()
	t.Setenv("MANGILE_UPLOADS", t.TempDir())
}

func TestRunVersion(t *testing.T) {
	withArgs(t, []string{"mangile", "version"})
	if got := run(); got != 0 {
		t.Errorf("version çıkış kodu 0 olmalı: %d", got)
	}
}

func TestRunVersionFlag(t *testing.T) {
	withArgs(t, []string{"mangile", "--version"})
	if got := run(); got != 0 {
		t.Errorf("--version çıkış kodu 0 olmalı: %d", got)
	}
}

func TestRunHelp(t *testing.T) {
	withArgs(t, []string{"mangile", "help"})
	if got := run(); got != 0 {
		t.Errorf("help çıkış kodu 0 olmalı: %d", got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	withArgs(t, []string{"mangile", "yokboylebirsey"})
	if got := run(); got == 0 {
		t.Error("bilinmeyen komut sıfır dışı kod vermeli")
	}
}

func TestRunUnknownFlag(t *testing.T) {
	withArgs(t, []string{"mangile", "--yokboylebirsey"})
	if got := run(); got == 0 {
		t.Error("bilinmeyen bayrak sıfır dışı kod vermeli")
	}
}

func TestRunDryRunWeb(t *testing.T) {
	withUploads(t)
	withArgs(t, []string{"mangile", "--dry-run", "web"})
	if got := run(); got != 0 {
		t.Errorf("dry-run web çıkış kodu 0 olmalı: %d", got)
	}
}

func TestRunDoctorEmpty(t *testing.T) {
	withUploads(t)
	withArgs(t, []string{"mangile", "doctor"})
	if got := run(); got != 0 {
		t.Errorf("boş dizinde doctor 0 vermeli: %d", got)
	}
}

func TestRunImportCSVdryRun(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "veri.csv")
	if err := os.WriteFile(csvPath, []byte("data,baslik,text\nCilt 1 Bölüm 1,Merhaba,dünya\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withUploads(t)
	withArgs(t, []string{"mangile", "--dry-run", "import", "csv", csvPath, "Deneme"})
	if got := run(); got != 0 {
		t.Errorf("dry-run import çıkış kodu 0 olmalı: %d", got)
	}
}

func TestRunImportUsage(t *testing.T) {
	withUploads(t)
	withArgs(t, []string{"mangile", "import"})
	if got := run(); got == 0 {
		t.Error("alt komutsuz import sıfır dışı kod vermeli")
	}
}
