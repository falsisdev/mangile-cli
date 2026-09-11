package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mangile-cli/internal/constants"
	"mangile-cli/internal/formats"
	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

type doctorIssue struct {
	where string
	what  string
}

func (a *App) Doctor(ctx context.Context) error {
	tui.PrintTitle("Tutarlılık Taraması")
	var issues []doctorIssue
	var warnings []doctorIssue

	series, err := uploads.DiscoverSeries(a.uploadsDir())
	if err != nil {
		return err
	}
	if len(series) == 0 {
		tui.PrintWarn("Taranacak seri yok: %s", a.uploadsDir())
		return nil
	}
	for _, s := range series {
		base := filepath.Base(s.Dir)
		if _, err := os.Stat(filepath.Join(s.Dir, "config.yaml")); os.IsNotExist(err) {
			issues = append(issues, doctorIssue{base, "config.yaml yok"})
			continue
		}
		if s.Config.MalID <= 0 {
			issues = append(issues, doctorIssue{base, "myAnimeListId tanımlı değil"})
		}
		if s.Config.Type != "manga" && s.Config.Type != "lightNovel" {
			issues = append(issues, doctorIssue{base, fmt.Sprintf("geçersiz tür: %q", s.Config.Type)})
		}
		if s.Config.UploadStatus != "" && !constants.IsValidStatus(s.Config.UploadStatus) {
			issues = append(issues, doctorIssue{base, fmt.Sprintf("geçersiz uploadStatus: %q", s.Config.UploadStatus)})
		}
		for _, tag := range s.Config.Tags {
			if !constants.IsValidTag(tag) {
				issues = append(issues, doctorIssue{base, fmt.Sprintf("kanonik dışı etiket: %q", tag)})
			}
		}
		entries, err := os.ReadDir(s.Dir)
		if err != nil {
			issues = append(issues, doctorIssue{base, fmt.Sprintf("dizin okunamadı: %v", err)})
			continue
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") || e.Name() == "config.yaml" || e.Name() == "scans.yaml" {
				continue
			}
			if !e.IsDir() && !formats.IsArchive(e.Name()) {
				continue
			}
			if e.IsDir() && !s.IsNovel() {
				pages := collectImagePages(filepath.Join(s.Dir, e.Name()))
				if len(pages) == 0 {
					warnings = append(warnings, doctorIssue{base, fmt.Sprintf("sayfasız bölüm klasörü: %s", e.Name())})
				}
			}
			if _, ok := parseChapterNumber(e.Name()); !ok {
				warnings = append(warnings, doctorIssue{base, fmt.Sprintf("numarası tespit edilemeyen bölüm: %s", e.Name())})
			}
		}
	}

	if a.Cfg.HasToken() && !a.isDry() {
		remoteIssues, remoteWarnings, err := a.doctorRemote(ctx, series)
		if err != nil {
			return err
		}
		issues = append(issues, remoteIssues...)
		warnings = append(warnings, remoteWarnings...)
	} else {
		tui.PrintDim("Sanity kontrolleri atlandı (token yok veya dry-run).")
	}

	printDoctorGroup("Sorunlar", issues)
	printDoctorGroup("Uyarılar", warnings)
	tui.PrintInfo("Özet: %d seri, %d sorun, %d uyarı", len(series), len(issues), len(warnings))
	if len(issues) > 0 {
		return fmt.Errorf("%d tutarlılık sorunu bulundu", len(issues))
	}
	tui.PrintSuccess("Tutarlılık tamam.")
	return nil
}

func printDoctorGroup(title string, items []doctorIssue) {
	if len(items) == 0 {
		return
	}
	tui.PrintInfo("---- %s (%d) ----", title, len(items))
	for _, it := range items {
		tui.PrintInfo("  • %s: %s", it.where, it.what)
	}
}

type remoteSeries struct {
	Type  string `json:"_type"`
	MalID int    `json:"myAnimeListId"`
	Title string `json:"title"`
}

func (a *App) doctorRemote(ctx context.Context, series []uploads.Series) ([]doctorIssue, []doctorIssue, error) {
	var issues []doctorIssue
	var warnings []doctorIssue
	var remote []remoteSeries
	groq := `*[_type in ["manga", "lightNovel"] && defined(myAnimeListId)]{_type, myAnimeListId, title}`
	if err := a.Client.Query(ctx, groq, nil, &remote); err != nil {
		return nil, nil, err
	}
	counts := map[string]int{}
	titles := map[string]string{}
	for _, r := range remote {
		key := fmt.Sprintf("%s/%d", r.Type, r.MalID)
		counts[key]++
		if _, ok := titles[key]; !ok {
			titles[key] = r.Title
		}
	}
	for key, n := range counts {
		if n > 1 {
			issues = append(issues, doctorIssue{"Sanity", fmt.Sprintf("MAL ID çakışması: %s (%d kayıt, örn. %q)", key, n, titles[key])})
		}
	}
	local := map[string]bool{}
	for _, s := range series {
		if s.Config.MalID > 0 {
			kind := "manga"
			if s.IsNovel() {
				kind = "lightNovel"
			}
			local[fmt.Sprintf("%s/%d", kind, s.Config.MalID)] = true
		}
	}
	orphan := 0
	for key := range counts {
		if !local[key] {
			orphan++
		}
	}
	if orphan > 0 {
		warnings = append(warnings, doctorIssue{"Sanity", fmt.Sprintf("yerelde karşılığı olmayan %d seri", orphan)})
	}
	for _, s := range series {
		if s.Config.MalID <= 0 {
			continue
		}
		kind := "manga"
		if s.IsNovel() {
			kind = "lightNovel"
		}
		ss, err := a.findSanitySeries(ctx, s.Config.MalID, kind)
		if err != nil {
			return nil, nil, err
		}
		if ss == nil {
			warnings = append(warnings, doctorIssue{s.Name(), fmt.Sprintf("Sanity'de karşılığı yok (MAL %d, %s)", s.Config.MalID, kind)})
		}
	}
	return issues, warnings, nil
}
