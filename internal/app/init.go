package app

import (
	"os"
	"path/filepath"

	"mangile-cli/internal/tui"
)

func (a *App) Init() error {
	tui.PrintTitle("Mangile Bazlı Yapılandırma")
	tui.PrintInfo("Üretim dizini: %s", a.uploadsDir())
	for _, dir := range []string{a.uploadsDir(), filepath.Join(a.uploadsDir(), ".state")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(a.uploadsDir(), "scans.yaml")); os.IsNotExist(err) {
		if err := os.WriteFile(filepath.Join(a.uploadsDir(), "scans.yaml"), []byte("scans: []\n"), 0o644); err != nil {
			return err
		}
	}
	tui.PrintSuccess("Dizin hazır. İçerik: uploads/<Seri>/Bölüm NN/ ...")
	tui.PrintInfo("Ortam değişkenleri: SANITY_TOKEN (zorunlu), MANGILE_PROJECT_ID (%s), MANGILE_DATASET (%s), MANGILE_UPLOADS", a.Cfg.ProjectID, a.Cfg.Dataset)
	tui.PrintInfo("Seri eklemek için: uploads/<Seri>/config.yaml oluşturun (varsayılan şablon otomatik yazılır).")
	return nil
}
