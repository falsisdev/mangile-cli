package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mangile-cli/internal/constants"
	"mangile-cli/internal/sanity"
	"mangile-cli/internal/tui"
)

func (a *App) Sync(ctx context.Context) error {
	if err := a.requireToken(); err != nil {
		return err
	}
	tui.PrintTitle("Sanity → Yerel Eşitleme")
	series, err := a.pickSeries(ctx)
	if err != nil {
		return err
	}
	kind := "manga"
	if series.IsNovel() {
		kind = "lightNovel"
	}
	sanityID, err := a.resolveSeriesID(ctx, series, kind)
	if err != nil {
		return err
	}
	chapters, err := a.listChapters(ctx, sanityID, kind)
	if err != nil {
		return err
	}
	if len(chapters) == 0 {
		tui.PrintWarn("Sanity'de bölüm yok: %s", series.Name())
		return nil
	}
	local, err := a.localChapterNums(series.Dir, kind == "lightNovel")
	if err != nil {
		return err
	}
	pending, skipped := syncPlan(chapters, local)
	tui.PrintInfo("Sanity: %d bölüm, yerelde var: %d, indirilecek: %d", len(chapters), skipped, len(pending))
	if len(pending) == 0 {
		tui.PrintSuccess("Her şey güncel.")
		return nil
	}
	for i, c := range pending {
		if i >= 8 {
			tui.PrintDim("  … %d bölüm daha", len(pending)-8)
			break
		}
		tui.PrintDim("  + %s", c.display())
	}
	if a.isDry() {
		tui.PrintInfo("[dry-run] Hiçbir dosya yazılmadı.")
		return nil
	}
	if !tui.ConfirmOrAbort(fmt.Sprintf("%d bölüm indirilsin mi?", len(pending))) {
		return nil
	}
	done := 0
	for _, c := range pending {
		if kind == "lightNovel" {
			if err := a.downloadNovelChapter(ctx, series.Dir, c.ID); err != nil {
				tui.PrintError("  %s: %v", c.display(), err)
				continue
			}
		} else {
			if err := a.downloadMangaChapter(ctx, series.Dir, c.ID); err != nil {
				tui.PrintError("  %s: %v", c.display(), err)
				continue
			}
		}
		done++
		tui.PrintSuccess("  %s indirildi", c.display())
	}
	tui.PrintSuccess("%d/%d bölüm eşitlendi.", done, len(pending))
	return nil
}

func syncPlan(chapters []sanityChapter, local map[string]bool) (pending []sanityChapter, skipped int) {
	byNum := map[string]sanityChapter{}
	for _, c := range chapters {
		key := normalizeChapterNum(formatNum(c.Number))
		cur, ok := byNum[key]
		if !ok || (!strings.HasPrefix(c.ID, "drafts.") && strings.HasPrefix(cur.ID, "drafts.")) {
			byNum[key] = c
		}
	}
	for _, c := range byNum {
		if local[normalizeChapterNum(formatNum(c.Number))] {
			skipped++
			continue
		}
		pending = append(pending, c)
	}
	return pending, skipped
}

func (a *App) localChapterNums(dir string, novel bool) (map[string]bool, error) {
	out := map[string]bool{}
	if novel {
		dirs, err := a.scanNovelDirs(dir)
		if err != nil {
			return nil, err
		}
		for _, d := range dirs {
			ch, err := readNovelChapter(d)
			if err != nil {
				continue
			}
			out[normalizeChapterNum(formatNum(ch.Number))] = true
		}
		return out, nil
	}
	chapters, err := a.scanMangaChapters(dir)
	if err != nil {
		return nil, err
	}
	defer cleanupChapters(chapters)
	for _, c := range chapters {
		if c.NumberP {
			out[normalizeChapterNum(formatNum(c.Number))] = true
		}
	}
	return out, nil
}

type mangaDetail struct {
	Title  string   `json:"title"`
	Number float64  `json:"chapterNumber"`
	Volume int      `json:"volumeNumber"`
	Pages  []string `json:"pages"`
}

func (a *App) downloadMangaChapter(ctx context.Context, seriesDir, id string) error {
	var res []mangaDetail
	groq := `*[_id == $id][0..0]{title, chapterNumber, volumeNumber, "pages": pages[].asset->url}`
	if err := a.Client.Query(ctx, groq, map[string]any{"id": id}, &res); err != nil {
		return err
	}
	if len(res) == 0 || len(res[0].Pages) == 0 {
		return fmt.Errorf("sayfa bulunamadı")
	}
	detail := res[0]
	target := filepath.Join(seriesDir, "Bölüm "+formatNum(detail.Number))
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("klasör zaten var: %s", target)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	for i, url := range detail.Pages {
		data, err := downloadURL(ctx, url, 20<<20)
		if err != nil {
			return fmt.Errorf("sayfa %d: %w", i+1, err)
		}
		name := fmt.Sprintf("%03d%s", i+1, imageExtFromURL(url))
		if err := os.WriteFile(filepath.Join(target, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

type novelDetail struct {
	Title   string           `json:"title"`
	Number  float64          `json:"chapterNumber"`
	Volume  int              `json:"volumeNumber"`
	Content []map[string]any `json:"content"`
}

func (a *App) downloadNovelChapter(ctx context.Context, seriesDir, id string) error {
	var res []novelDetail
	groq := `*[_id == $id][0..0]{title, chapterNumber, volumeNumber, content}`
	if err := a.Client.Query(ctx, groq, map[string]any{"id": id}, &res); err != nil {
		return err
	}
	if len(res) == 0 {
		return fmt.Errorf("bölüm bulunamadı")
	}
	detail := res[0]
	name := "Bölüm " + formatNum(detail.Number)
	if detail.Volume > 0 {
		name = "Cilt " + strconv.Itoa(detail.Volume) + " " + name
	}
	target := filepath.Join(seriesDir, name)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("klasör zaten var: %s", target)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	text := sanity.TextFromPortableText(detail.Content)
	files := map[string]string{
		"content.txt": text,
		"data.txt":    novelDataLine(detail.Volume, detail.Number),
	}
	if detail.Title != "" {
		files["baslik.txt"] = detail.Title
	}
	for fname, content := range files {
		if err := os.WriteFile(filepath.Join(target, fname), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func novelDataLine(volume int, number float64) string {
	if volume > 0 {
		return "Cilt " + strconv.Itoa(volume) + " Bölüm " + formatNum(number)
	}
	return "Bölüm " + formatNum(number)
}

func imageExtFromURL(url string) string {
	lower := strings.ToLower(url)
	if idx := strings.Index(lower, "?"); idx >= 0 {
		lower = lower[:idx]
	}
	ext := filepath.Ext(lower)
	if constants.IsImageExt(ext) {
		return ext
	}
	return ".jpg"
}

func downloadURL(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mangile-cli")
	resp, err := coverClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("indirme hatası (%d)", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("boş içerik")
	}
	return data, nil
}
