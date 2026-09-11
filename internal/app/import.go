package app

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mangile-cli/internal/constants"
	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

var (
	importCiltRe  = regexp.MustCompile(`(?i)cilt\s*(\d+)`)
	importBolumRe = regexp.MustCompile(`(?i)b[oö]l[uü]m\s*(\d+)`)
	importOnsozRe = regexp.MustCompile(`(?i)(önsöz|prologue|prolog)`)
	importSonszRe = regexp.MustCompile(`(?i)(sonsöz|epilogue|epilog)`)
)

func (a *App) ImportMenu(ctx context.Context) error {
	var pick string
	if err := tui.SelectOne("İçe aktarma türü", []string{"CSV'den klasör üret", "Eski chapters[] dizisini taşı"}, &pick); err != nil {
		return err
	}
	if pick == "CSV'den klasör üret" {
		return a.ImportCSV(ctx, "", "")
	}
	return a.ImportMigrate(ctx)
}

func (a *App) ImportCSV(ctx context.Context, path, seriesName string) error {
	_ = ctx
	if path == "" {
		var p string
		if err := tui.PromptText("CSV dosya yolu (ör. veri.csv)", &p); err != nil {
			return err
		}
		path = strings.TrimSpace(p)
	}
	if seriesName == "" {
		var s string
		if err := tui.PromptText("Hedef seri dizin adı", &s); err != nil {
			return err
		}
		seriesName = strings.TrimSpace(s)
	}
	if path == "" || seriesName == "" {
		return fmt.Errorf("CSV yolu ve seri adı zorunlu")
	}
	rows, headers, err := readCSVRows(path)
	if err != nil {
		return err
	}
	plan, err := planCSVFolders(headers, rows)
	if err != nil {
		return err
	}
	if len(plan) == 0 {
		tui.PrintWarn("CSV'de işlenecek satır yok.")
		return nil
	}
	tui.PrintInfo("CSV: %d satır → %d klasör (%s)", len(rows), len(plan), seriesName)
	for i, p := range plan {
		if i >= 5 {
			tui.PrintDim("  … %d klasör daha", len(plan)-5)
			break
		}
		tui.PrintDim("  %s", p.folder)
	}
	if a.isDry() {
		tui.PrintInfo("[dry-run] Hiçbir dosya yazılmadı.")
		return nil
	}
	if !tui.ConfirmOrAbort(fmt.Sprintf("%d klasör oluşturulsun mu?", len(plan))) {
		return nil
	}
	target := filepath.Join(a.uploadsDir(), seriesName)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	if err := uploads.WriteTemplateConfig(target); err != nil {
		return err
	}
	wrote := 0
	for _, p := range plan {
		dir := filepath.Join(target, p.folder)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tui.PrintWarn("Klasör açılamadı (%s): %v", p.folder, err)
			continue
		}
		for name, content := range p.files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
				tui.PrintWarn("Dosya yazılamadı (%s): %v", name, err)
			}
		}
		wrote++
	}
	tui.PrintSuccess("%d klasör oluşturuldu: %s", wrote, target)
	tui.PrintDim("Sonraki adım: seri config.yaml dosyasına myAnimeListId yazıp 'mangile run' ile yükleyin.")
	return nil
}

type csvFolder struct {
	folder string
	files  map[string]string
}

func readCSVRows(path string) ([][]string, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("CSV açılamadı: %w", err)
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("başlık satırı okunamadı: %w", err)
	}
	var rows [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("satır okuma hatası: %w", err)
		}
		rows = append(rows, record)
	}
	return rows, headers, nil
}

func planCSVFolders(headers []string, rows [][]string) ([]csvFolder, error) {
	dataIdx, titleIdx := -1, -1
	for i, h := range headers {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "data":
			dataIdx = i
		case "baslik", "title":
			if titleIdx == -1 {
				titleIdx = i
			}
		}
	}
	if dataIdx == -1 || titleIdx == -1 {
		return nil, fmt.Errorf("gerekli sütunlar yok ('data' ve 'baslik')")
	}
	var plan []csvFolder
	used := map[string]int{}
	for n, record := range rows {
		dataVal := csvCell(record, dataIdx)
		titleVal := strings.TrimSpace(csvCell(record, titleIdx))
		ciltNo := padThree(firstGroup(importCiltRe, dataVal))
		bolumNo := padThree(firstGroup(importBolumRe, dataVal))
		raw := fmt.Sprintf("Cilt %s Bölüm %s - %s", ciltNo, bolumNo, titleVal)
		folder := sanitizeCSVName(raw)
		if folder == "" {
			folder = fmt.Sprintf("isimsiz-klasor-%d", n+1)
		}
		if k := used[folder]; k > 0 {
			folder = fmt.Sprintf("%s-%d", folder, k+1)
		}
		used[folder]++
		files := map[string]string{}
		for i, colValue := range record {
			name := sanitizeCSVName(strings.TrimSpace(headers[i]))
			if name == "" {
				name = fmt.Sprintf("sutun-%d", i+1)
			}
			files[name+".txt"] = colValue
		}
		plan = append(plan, csvFolder{folder: folder, files: files})
	}
	return plan, nil
}

func csvCell(record []string, idx int) string {
	if idx < 0 || idx >= len(record) {
		return ""
	}
	return record[idx]
}

func firstGroup(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	return ""
}

func padThree(val string) string {
	if num, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
		return fmt.Sprintf("%03d", num)
	}
	if strings.TrimSpace(val) == "" {
		return "000"
	}
	return val
}

func sanitizeCSVName(name string) string {
	name = strings.TrimSpace(name)
	for invalid, replacement := range map[string]string{
		"/": "_", "\\": "_", ":": " -", "*": "_",
		"?": "_", "\"": "_", "<": "_", ">": "_", "|": "_",
	} {
		name = strings.ReplaceAll(name, invalid, replacement)
	}
	return strings.TrimSpace(name)
}

type legacyNovel struct {
	ID       string `json:"_id"`
	Title    string `json:"title"`
	MalID    int    `json:"myAnimeListId"`
	Chapters []struct {
		Title   string          `json:"title"`
		Source  map[string]any  `json:"source"`
		Content json.RawMessage `json:"content"`
	} `json:"chapters"`
}

func (a *App) ImportMigrate(ctx context.Context) error {
	if err := a.requireToken(); err != nil {
		return err
	}
	tui.PrintTitle("Eski Bölümleri Taşı (chapters[] → novelChapter)")
	var novels []legacyNovel
	groq := `*[_type == "lightNovel" && defined(chapters)]{_id, title, myAnimeListId, chapters[]{title, source, content}}`
	if err := a.Client.Query(ctx, groq, nil, &novels); err != nil {
		return err
	}
	if len(novels) == 0 {
		tui.PrintInfo("Taşınacak eski bölüm dizisi yok.")
		return nil
	}
	total := 0
	for _, n := range novels {
		total += len(n.Chapters)
	}
	tui.PrintInfo("%d seride %d gömülü bölüm bulundu.", len(novels), total)
	if a.isDry() {
		for _, n := range novels {
			tui.PrintInfo("  [dry-run] %s → %d novelChapter taslağı", n.Title, len(n.Chapters))
		}
		return nil
	}
	if !tui.ConfirmOrAbort("Taslak olarak taşınsın mı? (Eski chapters[] dizisine dokunulmaz)") {
		return nil
	}
	journal := uploads.NewJournal("", "lightNovel")
	if err := journal.Save(a.uploadsDir()); err != nil {
		return err
	}
	defer func() { _ = journal.Save(a.uploadsDir()) }()
	moved := 0
	for _, n := range novels {
		if n.MalID <= 0 {
			tui.PrintWarn("Atlandı (%s): myAnimeListId yok, deterministik ID üretilemez", n.Title)
			continue
		}
		for _, ch := range n.Chapters {
			vol, num, clean := parseMigrateTitle(ch.Title)
			finalID := chapterID("novelChapter", n.MalID, vol, formatNum(float64(num)))
			draftID := "drafts." + finalID
			doc := map[string]any{
				"_id":           draftID,
				"_type":         "novelChapter",
				"lightNovel":    ref(n.ID),
				"chapterNumber": num,
			}
			if vol > 0 {
				doc["volumeNumber"] = vol
			}
			if clean != "" {
				doc["title"] = clean
			}
			if len(ch.Source) > 0 {
				doc["source"] = ch.Source
			}
			if len(ch.Content) > 0 {
				var content []any
				if err := json.Unmarshal(ch.Content, &content); err == nil {
					doc["content"] = content
				}
			}
			if _, err := a.Client.CreateOrReplace(ctx, doc); err != nil {
				tui.PrintError("Taşınamadı (%s): %v", ch.Title, err)
				continue
			}
			journal.CreatedDocs = append(journal.CreatedDocs, draftID)
			moved++
		}
		tui.PrintInfo("  %s: bölümler taşındı", n.Title)
	}
	_ = journal.Save(a.uploadsDir())
	tui.PrintSuccess("%d bölüm taslak olarak taşındı. İnceleyip 'mangile publish' ile yayınlayın.", moved)
	tui.PrintWarn("Eski chapters[] dizileri duruyor; Studio'dan doğrulayıp temizleyin.")
	return nil
}

func parseMigrateTitle(title string) (vol int, num int, clean string) {
	if m := importCiltRe.FindStringSubmatch(title); len(m) > 1 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			vol = v
		}
	}
	switch {
	case importBolumRe.MatchString(title):
		m := importBolumRe.FindStringSubmatch(title)
		num, _ = strconv.Atoi(m[1])
	case importOnsozRe.MatchString(title):
		num = constants.NovelChapterPrologue
	case importSonszRe.MatchString(title):
		num = constants.NovelChapterEpilogue
	default:
		num = constants.NovelChapterExtra
	}
	if idx := strings.Index(title, ":"); idx >= 0 {
		clean = strings.TrimSpace(title[idx+1:])
	} else if vol == 0 {
		clean = strings.TrimSpace(title)
	}
	return vol, num, clean
}
