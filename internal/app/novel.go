package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"mangile-cli/internal/constants"
	"mangile-cli/internal/sanity"
	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

type NovelChapter struct {
	Display string
	Path    string
	Number  float64
	NumberP bool
	Volume  int
	Title   string
	Text    string
	Images  []string
}

var (
	imageNumRe = regexp.MustCompile(`(\d+)`)
	prologueRe = regexp.MustCompile(`(?i)(önsöz|prologue|prolog)`)
	epilogueRe = regexp.MustCompile(`(?i)(sonsöz|epilogue|epilog)`)
	extraRe    = regexp.MustCompile(`(?i)^[\s\._-]*(ekstra|extra|yan\s*hikaye|side\s*story)`)
)

func (a *App) NovelUpload(ctx context.Context) error {
	if err := a.requireToken(); err != nil {
		return err
	}
	tui.PrintTitle("Light Novel Bölümü Yükle")
	series, err := a.pickSeries(ctx)
	if err != nil {
		return err
	}
	if !series.IsNovel() {
		tui.PrintWarn("Bu seri manga olarak işaretli; novel yüklemesi için tipi lightNovel yapın (config.yaml: type: lightNovel).")
		return nil
	}
	sanityID, err := a.resolveSeriesID(ctx, series, "lightNovel")
	if err != nil {
		return err
	}

	chapterPaths, err := a.scanNovelDirs(series.Dir)
	if err != nil {
		return err
	}
	if len(chapterPaths) == 0 {
		tui.PrintWarn("Bölüm bulunamadı: %s", series.Dir)
		return nil
	}

	var labels []string
	for i, p := range chapterPaths {
		labels = append(labels, fmt.Sprintf("%2d. %s", i+1, filepath.Base(p)))
	}
	var selected []string
	if err := tui.SelectMany("Yüklenecek bölümler", labels, &selected); err != nil {
		return err
	}
	if len(selected) == 0 {
		return nil
	}

	chosenSet := map[string]string{}
	for i, p := range chapterPaths {
		chosenSet[labels[i]] = p
	}

	journal := uploads.NewJournal(sanityID, "lightNovel")
	if !a.isDry() {
		if err := journal.Save(a.uploadsDir()); err != nil {
			return err
		}
		defer func() { _ = journal.Save(a.uploadsDir()) }()
	}

	var published []string
	for _, sel := range selected {
		dir, ok := chosenSet[sel]
		if !ok {
			continue
		}
		ch, err := readNovelChapter(dir)
		if err != nil {
			tui.PrintError("Bölüm okunamadı (%s): %v", filepath.Base(dir), err)
			continue
		}
		tui.PrintInfo("Bölüm %s (%d satır)", ch.Display, strings.Count(ch.Text, "\n"))
		tui.PrintDim("  %d illüstrasyon", len(ch.Images))
		if preview := firstLinePreview(ch.Text); preview != "" {
			tui.PrintDim("  İlk satır: %s", preview)
		}
		if a.isDry() {
			draftID := "(numara tespit edilemedi)"
			if ch.NumberP {
				draftID = "drafts." + chapterID("novelChapter", series.Config.MalID, seriesVolume(ch.Volume), formatNum(ch.Number))
			}
			tui.PrintInfo("  [dry-run] novelChapter taslağı oluşturulacak → %s", draftID)
			continue
		}
		draftID, ok, err := a.uploadNovelChapter(ctx, journal, series, sanityID, ch)
		if err != nil {
			tui.PrintError("Bölüm yüklenemedi: %v", err)
			continue
		}
		if !ok {
			continue
		}
		published = append(published, draftID)
		tui.PrintSuccess("  %s taslak olarak kaydedildi (%s)", ch.Display, draftID)
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
	return nil
}

func (a *App) scanNovelDirs(seriesDir string) ([]string, error) {
	entries, err := os.ReadDir(seriesDir)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dirs = append(dirs, filepath.Join(seriesDir, e.Name()))
	}
	sortNovelDirs(dirs)
	return dirs, nil
}

func sortNovelDirs(dirs []string) {
	sort.SliceStable(dirs, func(i, j int) bool {
		ni, _ := parseChapterNumber(filepath.Base(dirs[i]))
		nj, _ := parseChapterNumber(filepath.Base(dirs[j]))
		return ni < nj
	})
}

func readNovelChapter(dir string) (*NovelChapter, error) {
	ch := &NovelChapter{Path: dir}
	base := filepath.Base(dir)
	ch.Number, ch.NumberP = parseChapterNumber(base)
	ch.Volume, _ = parseVolume(base)
	ch.Title = cleanChapterTitle(base)

	if data, err := os.ReadFile(filepath.Join(dir, "data.txt")); err == nil {
		content := string(data)
		if v, ok := parseChapterNumber(content); ok {
			ch.Number, ch.NumberP = v, true
		}
		if v, ok := parseVolume(content); ok {
			ch.Volume = v
		}
	}
	if baslik, err := os.ReadFile(filepath.Join(dir, "baslik.txt")); err == nil {
		if t := strings.TrimSpace(string(baslik)); t != "" {
			ch.Title = t
		}
	}
	ch.Text = readLegacyText(dir)
	if ch.Text == "" {
		main := pickNovelMainFile(dir)
		if main == "" {
			return nil, fmt.Errorf("metin dosyası yok (text.txt veya tek bir .txt/.md beklenir): %s", dir)
		}
		data, err := os.ReadFile(filepath.Join(dir, main))
		if err != nil {
			return nil, err
		}
		ch.Text = string(data)
	}
	ch.Images = readImageURLs(dir)

	if !ch.NumberP {
		ch.applySpecialCodes(base)
	}
	if !ch.NumberP {
		return nil, fmt.Errorf("bölüm numarası tespit edilemedi: %s", base)
	}
	if ch.Title == "" {
		ch.Title = base
	}
	ch.Display = "Bölüm " + formatNum(ch.Number)
	if ch.Volume > 0 {
		ch.Display = "Cilt " + strconv.Itoa(ch.Volume) + " " + ch.Display
	}
	return ch, nil
}

func (ch *NovelChapter) applySpecialCodes(base string) {
	lower := strings.ToLower(base)
	switch {
	case prologueRe.MatchString(lower):
		ch.Number, ch.NumberP = constants.NovelChapterPrologue, true
	case epilogueRe.MatchString(lower):
		ch.Number, ch.NumberP = constants.NovelChapterEpilogue, true
	case extraRe.MatchString(lower):
		ch.Number, ch.NumberP = constants.NovelChapterExtra, true
	}
}

func readLegacyText(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "text.txt"))
	if err != nil {
		return ""
	}
	return string(data)
}

func firstLinePreview(text string) string {
	if text == "" {
		return ""
	}
	s := strings.TrimSpace(text)
	if idx := strings.IndexAny(s, "\r\n"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	if len(s) > 80 {
		s = s[:80] + "…"
	}
	return s
}

func pickNovelMainFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var candidates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		low := strings.ToLower(e.Name())
		if strings.HasPrefix(low, "baslik") || strings.HasPrefix(low, "data") || strings.HasPrefix(low, "image_") {
			continue
		}
		if strings.HasSuffix(low, ".txt") || strings.HasSuffix(low, ".md") {
			candidates = append(candidates, e.Name())
		}
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	if len(candidates) > 1 {
		sort.Strings(candidates)
		return candidates[len(candidates)-1]
	}
	return ""
}

func readImageURLs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	type img struct {
		idx int
		url string
	}
	var imgs []img
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(strings.ToLower(e.Name()), "image_") {
			continue
		}
		m := imageNumRe.FindString(e.Name())
		if m == "" {
			continue
		}
		idx, _ := strconv.Atoi(m)
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		url := strings.TrimSpace(string(data))
		if url != "" {
			imgs = append(imgs, img{idx: idx, url: url})
		}
	}
	sort.Slice(imgs, func(i, j int) bool { return imgs[i].idx < imgs[j].idx })
	urls := make([]string, 0, len(imgs))
	for _, im := range imgs {
		urls = append(urls, im.url)
	}
	return urls
}

func (a *App) uploadNovelChapter(ctx context.Context, journal *uploads.Journal, series *uploads.Series, sanityID string, ch *NovelChapter) (string, bool, error) {
	if len(ch.Text) > constants.DocumentSizeLimit {
		tui.PrintWarn("Bölüm %s içerik %d bayt — Sanity doküman limiti %d aşıldı. Yine de yükleniyor (parçalama Faz 2'de eklenecek).", ch.Display, len(ch.Text), constants.DocumentSizeLimit)
	}
	vol := seriesVolume(ch.Volume)
	content := sanity.PortableTextFromText(ch.Text)
	for i, u := range ch.Images {
		label := fmt.Sprintf("[İllustrasyon %d](%s)", i+1, u)
		if b := sanity.ParagraphBlock(label); b != nil {
			content = append(content, b)
		}
	}
	if len(content) == 0 {
		content = append(content, sanity.ParagraphBlock(""))
	}

	draftID := "drafts." + chapterID("novelChapter", series.Config.MalID, vol, formatNum(ch.Number))
	doc := map[string]any{
		"_id":           draftID,
		"_type":         "novelChapter",
		"lightNovel":    ref(sanityID),
		"chapterNumber": ch.Number,
		"title":         ch.Title,
		"content":       content,
	}
	if vol > 0 {
		doc["volumeNumber"] = vol
	}
	if series.Config.DefaultScanID != "" {
		doc["source"] = ref(series.Config.DefaultScanID)
		if err := a.syncScanTitle(ctx, journal, series.Config.DefaultScanID, sanityID); err != nil {
			tui.PrintWarn("scan.titles senkronu başarısız: %v", err)
		}
	}

	if _, err := a.Client.CreateOrReplace(ctx, doc); err != nil {
		return "", false, err
	}
	journal.CreatedDocs = append(journal.CreatedDocs, draftID)
	return draftID, true, nil
}
