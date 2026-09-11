package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

type sanityChapter struct {
	ID     string   `json:"_id"`
	Rev    string   `json:"_rev"`
	Type   string   `json:"_type"`
	Title  string   `json:"title"`
	Number float64  `json:"chapterNumber"`
	Volume int      `json:"volumeNumber"`
	Assets []string `json:"assets"`
}

func (c sanityChapter) display() string {
	draft := ""
	if strings.HasPrefix(c.ID, "drafts.") {
		draft = " [taslak]"
	}
	label := "Bölüm " + formatNum(c.Number)
	if c.Volume > 0 {
		label = "Cilt " + strconv.Itoa(c.Volume) + " " + label
	}
	if c.Title != "" {
		label += " — " + c.Title
	}
	return label + draft
}

func (a *App) listChapters(ctx context.Context, seriesID, kind string) ([]sanityChapter, error) {
	var res []sanityChapter
	var groq string
	if kind == "lightNovel" {
		groq = `*[_type == "novelChapter" && lightNovel._ref == $id] | order(volumeNumber asc, chapterNumber asc) {
			_id, _rev, _type, title, chapterNumber, volumeNumber, "assets": []
		}`
	} else {
		groq = `*[_type == "mangaChapter" && manga._ref == $id] | order(volumeNumber asc, chapterNumber asc) {
			_id, _rev, _type, title, chapterNumber, volumeNumber, "assets": pages[].asset._ref
		}`
	}
	if err := a.Client.Query(ctx, groq, map[string]any{"id": seriesID}, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (a *App) ChapterManage(ctx context.Context, action string) error {
	if err := a.requireToken(); err != nil {
		return err
	}
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
	if a.isDry() {
		tui.PrintInfo("[dry-run] %s bölümleri listelenecek → %s", series.Name(), sanityID)
		tui.PrintInfo("[dry-run] İşlem: %s (hiçbir şey yazılmaz)", chapterActionLabel(action))
		return nil
	}
	chapters, err := a.listChapters(ctx, sanityID, kind)
	if err != nil {
		return err
	}
	if len(chapters) == 0 {
		tui.PrintWarn("Sanity'de bölüm yok: %s", series.Name())
		return nil
	}
	if action == "list" {
		tui.PrintInfo("---- Bölümler: %s (%d) ----", series.Name(), len(chapters))
		for _, c := range chapters {
			tui.PrintInfo("  • %s  (%s)", c.display(), c.ID)
		}
		return nil
	}
	if action == "" {
		var pick string
		if err := tui.SelectOne("Bölüm işlemi", []string{"Bölüm düzenle", "Bölüm sil"}, &pick); err != nil {
			return err
		}
		if pick == "Bölüm düzenle" {
			action = "edit"
		} else {
			action = "delete"
		}
	}
	ch := pickChapter(chapters)
	if ch == nil {
		return nil
	}
	switch action {
	case "edit":
		return a.editChapter(ctx, series, ch)
	case "delete":
		return a.deleteChapter(ctx, series, ch)
	default:
		return fmt.Errorf("bilinmeyen bölüm işlemi: %s", action)
	}
}

func chapterTwinIDs(id string) []string {
	twin := strings.TrimPrefix(id, "drafts.")
	if twin == id {
		return []string{id, "drafts." + id}
	}
	return []string{id, twin}
}

func chapterActionLabel(action string) string {
	switch action {
	case "list":
		return "listele"
	case "edit":
		return "düzenle"
	case "delete":
		return "sil"
	default:
		return "seç"
	}
}

func pickChapter(chapters []sanityChapter) *sanityChapter {
	var labels []string
	byLabel := map[string]int{}
	for i, c := range chapters {
		label := fmt.Sprintf("%2d. %s", i+1, c.display())
		labels = append(labels, label)
		byLabel[label] = i
	}
	var selected string
	if err := tui.SelectOne("Bölüm seçin", labels, &selected); err != nil {
		return nil
	}
	idx, ok := byLabel[selected]
	if !ok {
		return nil
	}
	return &chapters[idx]
}

func (a *App) editChapter(ctx context.Context, series *uploads.Series, ch *sanityChapter) error {
	tui.PrintTitle("Bölüm Düzenle")
	tui.PrintInfo("Bölüm: %s", ch.display())
	tui.PrintDim("Boş bırakılan alan korunur. Yalnızca başlık ve cilt düzenlenir.")
	var title string
	if err := tui.PromptText(fmt.Sprintf("Başlık (mevcut: %s)", ch.Title), &title); err != nil {
		return err
	}
	var volumeStr string
	if err := tui.PromptText(fmt.Sprintf("Cilt (mevcut: %d)", ch.Volume), &volumeStr); err != nil {
		return err
	}
	patch := map[string]any{}
	if t := strings.TrimSpace(title); t != "" {
		patch["title"] = t
	}
	if v := strings.TrimSpace(volumeStr); v != "" {
		vol, err := strconv.Atoi(v)
		if err != nil || vol < 0 {
			return fmt.Errorf("cilt sayısı geçersiz: %s", v)
		}
		patch["volumeNumber"] = vol
	}
	if len(patch) == 0 {
		tui.PrintInfo("Değişiklik yok.")
		return nil
	}
	if a.isDry() {
		tui.PrintInfo("[dry-run] %s yamalanacak: %v", ch.ID, patch)
		return nil
	}
	if !tui.ConfirmOrAbort(fmt.Sprintf("'%s' güncellensin mi?", ch.display())) {
		return nil
	}
	before := map[string]any{"title": ch.Title, "volumeNumber": ch.Volume}
	if err := a.Client.Patch(ctx, ch.ID, patch); err != nil {
		return err
	}
	journal := uploads.NewJournal(series.Config.SanityID, series.Config.Type)
	journal.PatchedDocs = append(journal.PatchedDocs, uploads.JournalPatch{ID: ch.ID, Rev: ch.Rev, Before: before})
	if err := journal.Save(a.uploadsDir()); err != nil {
		tui.PrintWarn("Günlük kaydedilemedi: %v", err)
	}
	tui.PrintSuccess("Bölüm güncellendi (%s). Geri almak için 'mangile rollback' kullanın.", ch.ID)
	return nil
}

func (a *App) deleteChapter(ctx context.Context, series *uploads.Series, ch *sanityChapter) error {
	tui.PrintTitle("Bölüm Sil")
	tui.PrintWarn("Bölüm: %s (%s)", ch.display(), ch.ID)
	tui.PrintWarn("Taslak ve yayınlanmış kopya birlikte silinir. Kullanılmayan görseller temizlenir.")
	if a.isDry() {
		tui.PrintInfo("[dry-run] %s silinecek, %d asset yetim kontrolünden geçecek", ch.ID, len(ch.Assets))
		return nil
	}
	if !tui.ConfirmOrAbort("Bölüm kalıcı olarak silinsin mi?") {
		return nil
	}
	ids := chapterTwinIDs(ch.ID)
	existing, err := a.filterExisting(ctx, ids)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		tui.PrintWarn("Silinecek kayıt bulunamadı.")
		return nil
	}
	if err := a.Client.Delete(ctx, existing); err != nil {
		return err
	}
	removed := a.deleteOrphanAssets(ctx, ch.Assets)
	journal := uploads.NewJournal(series.Config.SanityID, series.Config.Type)
	journal.DeletedDocs = append(journal.DeletedDocs, uploads.DeletedDoc{
		ID:    ch.ID,
		Type:  ch.Type,
		Title: ch.Title,
		Snapshot: map[string]any{
			"chapterNumber": ch.Number,
			"volumeNumber":  ch.Volume,
		},
	})
	journal.Assets = append(journal.Assets, ch.Assets...)
	if err := journal.Save(a.uploadsDir()); err != nil {
		tui.PrintWarn("Günlük kaydedilemedi: %v", err)
	}
	tui.PrintSuccess("Bölüm silindi (%d kayıt). Temizlenen yetim asset: %d", len(existing), removed)
	return nil
}
