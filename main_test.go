package main

import (
	"os"
	"path/filepath"
	"testing"

	"mangile-cli/internal/uploads"
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

func TestRunRollbackUnknownID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MANGILE_UPLOADS", dir)
	t.Setenv("SANITY_TOKEN", "sahte")
	j := uploads.NewJournal("seri-1", "manga")
	if err := j.Save(dir); err != nil {
		t.Fatal(err)
	}
	withArgs(t, []string{"mangile", "rollback", "yok-123"})
	if got := run(); got == 0 {
		t.Error("bilinmeyen günlük ID'si sıfır dışı kod vermeli")
	}
}

func TestRunRollbackList(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MANGILE_UPLOADS", dir)
	t.Setenv("SANITY_TOKEN", "sahte")
	j := uploads.NewJournal("seri-1", "manga")
	if err := j.Save(dir); err != nil {
		t.Fatal(err)
	}
	withArgs(t, []string{"mangile", "rollback", "--list"})
	if got := run(); got != 0 {
		t.Errorf("--list 0 vermeli: %d", got)
	}
}

func TestRunUpgradeDryRun(t *testing.T) {
	withUploads(t)
	withArgs(t, []string{"mangile", "--dry-run", "upgrade"})
	if got := run(); got != 0 {
		t.Errorf("dry-run upgrade 0 vermeli: %d", got)
	}
}
