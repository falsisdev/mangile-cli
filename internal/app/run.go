package app

import (
	"context"

	"mangile-cli/internal/tui"
)

func (a *App) Run(ctx context.Context) error {
	tui.PrintTitle("Mangile CLI")
	tui.PrintDim("Proje: %s | Veri seti: %s | Dizin: %s", a.Cfg.ProjectID, a.Cfg.Dataset, a.uploadsDir())
	if a.isDry() {
		tui.PrintWarn("--dry-run aktif: hiçbir şey yazılmaz, yalnızca plan gösterilir.")
	}
	if !a.Cfg.HasToken() {
		tui.PrintWarn("SANITY_TOKEN tanımlı değil — ağ işlemleri çalışmaz.")
	}

	actions := []string{
		"Seri durumunu göster (reconcile)",
		"Manga bölümü yükle",
		"Light novel bölümü yükle",
		"Yeni seri dizini oluştur",
		"Bölüm düzenle / sil",
		"Seri oluştur (Jikan destekli)",
		"Seri güncelle",
		"Web yükleyici (tarayıcı)",
		"Tutarlılık tara (doctor)",
		"İçe aktar (CSV / migrasyon)",
		"Taslakları yayınla",
		"Değişiklikleri geri al (rollback)",
		"Çıkış",
	}
	for {
		var pick string
		if err := tui.SelectOne("Yapılacak işlem", actions, &pick); err != nil {
			return err
		}
		switch pick {
		case actions[12]:
			tui.PrintInfo("Görüşürüz.")
			return nil
		case actions[0]:
			if err := a.SeriesList(ctx); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[1]:
			if err := a.MangaUpload(ctx); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[2]:
			if err := a.NovelUpload(ctx); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[3]:
			if err := a.CreateSeriesDir(ctx); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[4]:
			if err := a.ChapterManage(ctx, ""); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[5]:
			if tui.ConfirmOrAbort("Jikan'dan bilgi çekilsin mi?") {
				if err := a.CreateSeries(ctx, true); err != nil {
					tui.PrintError("%v", err)
				}
			} else if err := a.CreateSeries(ctx, false); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[6]:
			if tui.ConfirmOrAbort("Jikan'dan eksikler doldurulsun mu?") {
				if err := a.UpdateSeries(ctx, true); err != nil {
					tui.PrintError("%v", err)
				}
			} else if err := a.UpdateSeries(ctx, false); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[7]:
			if err := a.WebServe(ctx); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[8]:
			if err := a.Doctor(ctx); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[9]:
			if err := a.ImportMenu(ctx); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[10]:
			if err := a.PublishAll(ctx); err != nil {
				tui.PrintError("%v", err)
			}
		case actions[11]:
			if err := a.Rollback(ctx); err != nil {
				tui.PrintError("%v", err)
			}
		}
	}
}
