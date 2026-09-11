package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mangile-cli/internal/constants"
	"mangile-cli/internal/formats"
	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

type PageFile struct {
	Name string
	Path string
}

type MangaChapter struct {
	Display string
	Kind    string
	Path    string
	Number  float64
	NumberP bool
	Volume  int
	Title   string
	TempDir string
	Pages   []PageFile
	Empty   bool
}

func (a *App) MangaUpload(ctx context.Context) error {
	if err := a.requireToken(); err != nil {
		return err
	}
	tui.PrintTitle("Manga Bölümü Yükle")
	series, err := a.pickSeries(ctx)
	if err != nil {
		return err
	}
	if series.IsNovel() {
		tui.PrintWarn("Bu seri lightNovel olarak işaretli; manga yüklemesi uygun değil.")
		return nil
	}
	sanityID, err := a.resolveSeriesID(ctx, series, "manga")
	if err != nil {
		return err
	}

	chapters, err := a.scanMangaChapters(series.Dir)
	if err != nil {
		return err
	}
	if len(chapters) == 0 {
		tui.PrintWarn("Bölüm bulunamadı: %s (klasör veya cbz/zip/tar.gz beklenir)", series.Dir)
		return nil
	}

	filtered := skipEmptyChapters(chapters)
	if len(filtered) == 0 {
		tui.PrintWarn("Tüm bölümlerde görsel yok.")
		return nil
	}

	var labels []string
	for _, c := range filtered {
		labels = append(labels, fmt.Sprintf("%s (%d sayfa)", c.Display, len(c.Pages)))
	}
	var selected []string
	if err := tui.SelectMany("Yüklenecek bölümler", labels, &selected); err != nil {
		return err
	}
	if len(selected) == 0 {
		return nil
	}

	byLabel := map[string]*MangaChapter{}
	for i, c := range filtered {
		byLabel[labels[i]] = c
	}
	var chosen []*MangaChapter
	for _, sel := range selected {
		if c, ok := byLabel[sel]; ok {
			chosen = append(chosen, c)
		}
	}

	tui.PrintInfo("Yüklenecek %d bölüm:", len(chosen))
	for _, c := range chosen {
		tui.PrintDim("  • %s → %d sayfa", c.Display, len(c.Pages))
	}

	if !a.isDry() && !tui.ConfirmOrAbort("Sıra doğru mu? Yükleme başlasın mı?") {
		return nil
	}

	journal := uploads.NewJournal(sanityID, "manga")
	if !a.isDry() {
		if err := journal.Save(a.uploadsDir()); err != nil {
			return err
		}
		defer func() {
			_ = journal.Save(a.uploadsDir())
		}()
	}

	var published []string
	for _, c := range chosen {
		tui.PrintInfo("Bölüm %s işleniyor…", c.Display)
		if a.isDry() {
			tui.PrintInfo("  [dry-run] %d sayfa yüklenecek + mangaChapter taslağı oluşturulacak", len(c.Pages))
			continue
		}
		draftID, ok, err := a.uploadMangaChapter(ctx, journal, series, sanityID, c)
		if err != nil {
			tui.PrintError("  HATA: %v", err)
			continue
		}
		if !ok {
			continue
		}
		published = append(published, draftID)
		tui.PrintSuccess("  Bölüm %s taslak olarak kaydedildi (%s)", c.Display, draftID)
	}

	if !a.isDry() && len(published) > 0 {
		_ = journal.Save(a.uploadsDir())
		if tui.ConfirmOrAbort("Taslakları şimdi yayınlamak ister misiniz?") {
			if err := a.publishIDs(ctx, published); err != nil {
				tui.PrintError("Yayınlama hatası: %v", err)
			} else {
				tui.PrintSuccess("%d bölüm yayınlandı.", len(published))
			}
		} else {
			tui.PrintWarn("Taslaklar Studio'da incelenebilir. Yayınlamak için 'mangile publish' kullanın.")
		}
	}

	cleanupChapters(filtered)
	return nil
}

func (a *App) scanMangaChapters(seriesDir string) ([]*MangaChapter, error) {
	entries, err := os.ReadDir(seriesDir)
	if err != nil {
		return nil, err
	}
	var chapters []*MangaChapter
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		path := filepath.Join(seriesDir, e.Name())
		var ch *MangaChapter
		if e.IsDir() {
			ch = a.scanMangaFolder(path)
		} else if formats.IsArchive(e.Name()) {
			ch = a.scanMangaArchive(path)
		}
		if ch != nil {
			chapters = append(chapters, ch)
		}
	}
	sortChaptersByNumber(chapters)
	return chapters, nil
}

func (a *App) scanMangaFolder(path string) *MangaChapter {
	base := filepath.Base(path)
	ch := &MangaChapter{Kind: "folder", Path: path}
	ch.Number, ch.NumberP = parseChapterNumber(base)
	ch.Volume, _ = parseVolume(base)
	ch.Title = cleanChapterTitle(base)
	if ci, ok := readComicInfo(path); ok {
		if n, err := parseComicNumber(ci.Number); err == nil && n > 0 {
			ch.Number, ch.NumberP = n, true
		}
		if v, err := parseComicVolume(ci.Volume); err == nil && v > 0 {
			ch.Volume = v
		}
		if ci.Title != "" {
			ch.Title = ci.Title
		}
	}
	ch.Pages = collectImagePages(path)
	ch.Display = chapterDisplay(ch)
	return ch
}

func (a *App) scanMangaArchive(path string) *MangaChapter {
	base := filepath.Base(path)
	ch := &MangaChapter{Kind: "archive", Path: path}
	ch.Number, ch.NumberP = parseChapterNumber(base)
	ch.Volume, _ = parseVolume(base)
	ch.Title = cleanChapterTitle(base)
	tmp, err := os.MkdirTemp("", "mangile-archive-")
	if err != nil {
		return ch
	}
	if err := formats.ExpandArchive(path, tmp); err != nil {
		tui.PrintWarn("Arşiv açılamadı (%s): %v", base, err)
		return nil
	}
	ch.TempDir = tmp
	ch.Pages = collectImagePages(tmp)
	ch.Display = chapterDisplay(ch)
	return ch
}

func collectImagePages(dir string) []PageFile {
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return nil
	}
	var entries []formats.Entry
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if constants.IsImageExt(strings.ToLower(filepath.Ext(d.Name()))) {
			entries = append(entries, formats.Entry{Name: d.Name(), Path: p})
		}
		return nil
	})
	result := formats.SortPages(entries)
	for _, w := range result.Warnings {
		tui.PrintWarn("  %s", w)
	}
	var pages []PageFile
	for _, e := range result.Entries {
		pages = append(pages, PageFile{Name: e.Name, Path: e.Path})
	}
	return pages
}

func readComicInfo(dir string) (formats.ComicInfoData, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "ComicInfo.xml"))
	if err != nil {
		return formats.ComicInfoData{}, false
	}
	return formats.ParseComicInfo(data)
}

func parseComicNumber(s string) (float64, error) {
	return parseFloatSafe(s)
}

func parseComicVolume(s string) (int, error) {
	return parseIntSafe(s)
}

func parseFloatSafe(s string) (float64, error) {
	var v float64
	_, err := fmt.Sscanf(strings.ReplaceAll(strings.TrimSpace(s), ",", "."), "%f", &v)
	return v, err
}

func parseIntSafe(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &v)
	return v, err
}

func chapterDisplay(ch *MangaChapter) string {
	label := filepath.Base(ch.Path)
	num := ""
	if ch.NumberP {
		num = "Bölüm " + formatNum(ch.Number)
	}
	return fmt.Sprintf("%s (%s)", label, num)
}

func formatNum(f float64) string {
	return strings.TrimSuffix(fmt.Sprintf("%g", f), ".0")
}

func skipEmptyChapters(chapters []*MangaChapter) []*MangaChapter {
	out := make([]*MangaChapter, 0, len(chapters))
	for _, c := range chapters {
		if len(c.Pages) > 0 {
			out = append(out, c)
		}
	}
	return out
}

func sortChaptersByNumber(chapters []*MangaChapter) {
	for i := 1; i < len(chapters); i++ {
		for j := i; j > 0 && chapterLess(chapters[j], chapters[j-1]); j-- {
			chapters[j], chapters[j-1] = chapters[j-1], chapters[j]
		}
	}
}

func chapterLess(a, b *MangaChapter) bool {
	if a.NumberP && b.NumberP && a.Number != b.Number {
		return a.Number < b.Number
	}
	return formats.NaturalLess(filepath.Base(a.Path), filepath.Base(b.Path))
}

func cleanupChapters(chapters []*MangaChapter) {
	for _, c := range chapters {
		if c.TempDir != "" {
			_ = os.RemoveAll(c.TempDir)
		}
	}
}

func (a *App) uploadMangaChapter(ctx context.Context, journal *uploads.Journal, series *uploads.Series, sanityID string, ch *MangaChapter) (string, bool, error) {
	if !ch.NumberP {
		tui.PrintWarn("Hata: bölüm numarası tespit edilemedi: %s", ch.Display)
		return "", false, errors.New("bölüm numarası tespit edilemedi")
	}
	vol := seriesVolume(ch.Volume)
	numStr := formatNum(ch.Number)
	finalID := chapterID("mangaChapter", series.Config.MalID, vol, numStr)
	draftID := "drafts." + finalID

	var assetRefs []map[string]any
	for _, page := range ch.Pages {
		content, err := os.ReadFile(page.Path)
		if err != nil {
			return "", false, err
		}
		assetID, err := a.Client.UploadAsset(ctx, content, page.Name, contentTypeFor(page.Name))
		if err != nil {
			return "", false, err
		}
		journal.Assets = append(journal.Assets, assetID)
		assetRefs = append(assetRefs, map[string]any{
			"_key":  "page_" + formatNum(float64(len(assetRefs)+1)),
			"_type": "image",
			"asset": map[string]any{"_type": "reference", "_ref": assetID},
		})
		tui.PrintInfo("  + %s → %s", page.Name, assetID)
	}

	doc := map[string]any{
		"_id":           draftID,
		"_type":         "mangaChapter",
		"manga":         ref(sanityID),
		"chapterNumber": ch.Number,
		"pages":         assetRefs,
	}
	if vol > 0 {
		doc["volumeNumber"] = vol
	}
	if ch.Title != "" {
		doc["title"] = ch.Title
	}
	if series.Config.DefaultScanID != "" {
		doc["source"] = ref(series.Config.DefaultScanID)
	}

	if _, err := a.Client.CreateOrReplace(ctx, doc); err != nil {
		return "", false, err
	}
	journal.CreatedDocs = append(journal.CreatedDocs, draftID)

	if series.Config.DefaultScanID != "" {
		if err := a.syncScanTitle(ctx, journal, series.Config.DefaultScanID, sanityID, false); err != nil {
			tui.PrintWarn("scan.titles senkronu başarısız: %v", err)
		}
	}
	return draftID, true, nil
}

func seriesVolume(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func contentTypeFor(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
