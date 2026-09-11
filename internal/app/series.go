package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

func (a *App) SeriesList(ctx context.Context) error {
	if err := a.requireToken(); err != nil {
		return err
	}
	series, err := uploads.DiscoverSeries(a.uploadsDir())
	if err != nil {
		return err
	}
	if len(series) == 0 {
		tui.PrintWarn("uploads dizininde seri yok. 'Yeni seri dizini oluştur' ile ekleyin.")
		return nil
	}
	tui.PrintInfo("---- Seri Durumu (%d) ----", len(series))
	for _, s := range series {
		kind := "manga"
		if s.IsNovel() {
			kind = "lightNovel"
		}
		statusLine := fmt.Sprintf("%-30s [%s]", s.Name(), kind)
		if s.Config.MalID > 0 {
			statusLine += fmt.Sprintf(" | MAL: %d", s.Config.MalID)
		}
		if s.Config.SanityID != "" {
			statusLine += " | ID: " + s.Config.SanityID
		}
		matched := "bulunamadı"
		if a.isDry() {
			matched = "atlandı (dry-run)"
		} else {
			id := s.Config.SanityID
			var ss *sanitySeries
			if id != "" {
				ss, _ = a.fetchSeriesByID(ctx, id)
			}
			if ss == nil {
				ss, _ = a.findSanitySeries(ctx, s.Config.MalID, s.Config.Type)
			}
			if ss != nil {
				matched = "bulundu: " + ss.ID + " (" + ss.UploadStatus + ")"
				if s.Config.SanityID != ss.ID {
					s.Config.SanityID = ss.ID
					_ = uploads.SaveSeriesConfig(s.Dir, s.Config)
					matched += " → config.yaml'ye yazıldı"
				}
			}
		}
		tui.PrintInfo("%s → Sanity: %s", statusLine, matched)
	}
	return nil
}

func (a *App) CreateSeriesDir(ctx context.Context) error {
	var name string
	if err := tui.PromptText("Seri dizini adı (ör. Mushoku Tensei)", &name); err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	dir := a.uploadsDir() + "/" + name
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	cfg := uploads.SeriesConfig{
		MalID:        askInt("MyAnimeList ID"),
		Type:         askType(),
		Title:        name,
		UploadStatus: "uploading",
	}
	for {
		var tag string
		if err := tui.PromptText("Tür etiketi (boş bırakılırsa geçilir)", &tag); err != nil {
			tag = ""
		}
		tag = strings.TrimSpace(tag)
		if tag == "" {
			break
		}
		cfg.Tags = append(cfg.Tags, tag)
		if len(cfg.Tags) >= 5 {
			break
		}
	}
	if err := uploads.SaveSeriesConfig(dir, cfg); err != nil {
		return err
	}
	tui.PrintSuccess("Seri dizini oluşturuldu: %s", dir)
	return nil
}

func askInt(title string) int {
	var v int
	tui.PromptInt(title, &v)
	return v
}

func askType() string {
	var t string
	opts := []string{"manga", "lightNovel"}
	tui.SelectOne("Seri türü", opts, &t)
	return t
}
