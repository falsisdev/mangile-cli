package app

import (
	"os"
	"path/filepath"
	"strings"

	"mangile-cli/internal/cfg"
	"mangile-cli/internal/tui"
)

func (a *App) Init() error {
	tui.PrintTitle("Mangile Bazlı Yapılandırma")
	tui.PrintInfo("Üretim dizini: %s", a.uploadsDir())
	if err := a.scaffoldUploads(); err != nil {
		return err
	}
	tui.PrintSuccess("Dizin hazır. İçerik: <Seri>/Bölüm NN/ ...")
	tui.PrintInfo("Ortam değişkenleri: SANITY_TOKEN (zorunlu), MANGILE_PROJECT_ID (%s), MANGILE_DATASET (%s), MANGILE_UPLOADS", a.Cfg.ProjectID, a.Cfg.Dataset)
	tui.PrintInfo("Seri eklemek için: <Seri>/config.yaml oluşturun (varsayılan şablon otomatik yazılır).")
	return nil
}

func (a *App) RunSetupWizard() error {
	tui.PrintTitle("Mangile İlk Kurulum Sihirbazı")
	tui.PrintInfo("İçerik dizini nerede tutulsun? (Enter = %s)", cfg.DefaultUploadsPath())
	var choice string
	if err := tui.PromptText("İçerik dizini", &choice); err != nil {
		return err
	}
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = cfg.DefaultUploadsPath()
	}
	abs, err := filepath.Abs(choice)
	if err != nil {
		return err
	}
	if err := cfg.SaveUserConfig(cfg.UserConfig{UploadsPath: abs}); err != nil {
		tui.PrintWarn("Yapılandırma kaydedilemedi: %v (yola devam)", err)
	}
	a.uploads = abs
	tui.PrintInfo("İçerik dizini: %s", abs)
	if err := a.scaffoldUploads(); err != nil {
		return err
	}
	tui.PrintSuccess("Kurulum tamamlandı. Herhangi bir terminalden 'mangile' çalıştırabilirsiniz.")
	return nil
}

func (a *App) scaffoldUploads() error {
	for _, dir := range []string{a.uploadsDir(), filepath.Join(a.uploadsDir(), ".state")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	scansPath := filepath.Join(a.uploadsDir(), "scans.yaml")
	if _, err := os.Stat(scansPath); os.IsNotExist(err) {
		if err := os.WriteFile(scansPath, []byte("scans: []\n"), 0o644); err != nil {
			return err
		}
	}
	return nil
}
