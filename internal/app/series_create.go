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
	"time"

	"mangile-cli/internal/constants"
	"mangile-cli/internal/formats"
	"mangile-cli/internal/jikan"
	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

var coverClient = &http.Client{Timeout: 60 * time.Second}

func (a *App) CreateSeries(ctx context.Context, fetch bool) error {
	if err := a.requireToken(); err != nil {
		return err
	}
	tui.PrintTitle("Seri Oluştur")
	malID := askInt("MyAnimeList ID")
	if malID <= 0 {
		return fmt.Errorf("geçerli bir MyAnimeList ID girin")
	}
	kind := askType()
	var title, description, imageURL, jikanKind string
	if fetch {
		tui.PrintInfo("Jikan'dan alınıyor: MAL %d…", malID)
		meta, err := jikan.FetchManga(malID)
		if err != nil {
			return err
		}
		title, description, imageURL, jikanKind = meta.Title, meta.Synopsis, meta.ImageURL, meta.Kind
		tui.PrintInfo("Bulundu: %s (%s)", title, jikanKind)
		if description != "" {
			tui.PrintDim("  %s", firstLinePreview(description))
		}
		if !tui.ConfirmOrAbort("Bu bilgiyle devam edilsin mi?") {
			return nil
		}
	} else {
		var t string
		if err := tui.PromptText("Seri başlığı", &t); err != nil {
			return err
		}
		title = strings.TrimSpace(t)
		if title == "" {
			return fmt.Errorf("başlık boş olamaz")
		}
	}
	existing, err := a.findSanitySeries(ctx, malID, kind)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("MAL ID %d zaten Sanity'de kayıtlı (%s). Güncellemek için 'mangile update' kullanın", malID, existing.ID)
	}
	docID := kind + "-" + itoa(malID)
	draftID := "drafts." + docID
	doc := map[string]any{
		"_id":           draftID,
		"_type":         kind,
		"title":         title,
		"slug":          map[string]any{"_type": "slug", "current": formats.Slugify(title)},
		"myAnimeListId": malID,
		"uploadStatus":  constants.StatusUploading,
	}
	if kind == "manga" {
		if format := askFormat(); format != "" {
			doc["format"] = format
		}
	}
	if description != "" {
		doc["description"] = description
	}
	if a.isDry() {
		tui.PrintInfo("[dry-run] %s taslağı oluşturulacak (%s)", kind, draftID)
		tui.PrintInfo("[dry-run] Yerel dizin + config.yaml yazılacak: %s", filepath.Join(a.uploadsDir(), title))
		return nil
	}
	if imageURL != "" {
		assetID, err := a.uploadCoverImage(ctx, imageURL, title)
		if err != nil {
			tui.PrintWarn("Kapak yüklenemedi, kapaksız devam ediliyor: %v", err)
		} else {
			doc["coverImage"] = map[string]any{"_type": "image", "asset": ref(assetID)}
		}
	}
	if _, err := a.Client.CreateOrReplace(ctx, doc); err != nil {
		return err
	}
	journal := uploads.NewJournal(docID, kind)
	journal.CreatedDocs = append(journal.CreatedDocs, draftID)
	if err := journal.Save(a.uploadsDir()); err != nil {
		tui.PrintWarn("Günlük kaydedilemedi: %v", err)
	}
	dir := filepath.Join(a.uploadsDir(), title)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	local := uploads.SeriesConfig{SanityID: docID, MalID: malID, Type: kind, Title: title, UploadStatus: constants.StatusUploading}
	if description != "" {
		local.Description = description
	}
	if err := uploads.SaveSeriesConfig(dir, local); err != nil {
		return err
	}
	tui.PrintSuccess("Seri taslağı oluşturuldu (%s). Yerel dizin: %s", draftID, dir)
	if tui.ConfirmOrAbort("Taslak şimdi yayınlansın mı?") {
		if err := a.publishIDs(ctx, []string{draftID}); err != nil {
			tui.PrintError("Yayınlama hatası: %v", err)
		} else {
			tui.PrintSuccess("Seri yayınlandı.")
		}
	} else {
		tui.PrintWarn("Taslak Studio'da incelenebilir. Yayınlamak için 'mangile publish' kullanın.")
	}
	return nil
}

func (a *App) UpdateSeries(ctx context.Context, fetch bool) error {
	if err := a.requireToken(); err != nil {
		return err
	}
	tui.PrintTitle("Seri Güncelle")
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
	ss, err := a.fetchSeriesByID(ctx, sanityID)
	if err != nil {
		return err
	}
	if ss == nil {
		return fmt.Errorf("seri Sanity'de bulunamadı: %s", sanityID)
	}
	patch := map[string]any{}
	if fetch {
		tui.PrintInfo("Jikan'dan alınıyor: MAL %d…", ss.MalID)
		meta, err := jikan.FetchManga(ss.MalID)
		if err != nil {
			return err
		}
		if ss.Title == "" && meta.Title != "" {
			patch["title"] = meta.Title
		}
		if ss.Description == "" && meta.Synopsis != "" {
			patch["description"] = meta.Synopsis
		}
		if meta.ImageURL != "" {
			hasCover, err := a.seriesHasCover(ctx, sanityID)
			if err != nil {
				return err
			}
			if !hasCover {
				if a.isDry() {
					tui.PrintInfo("[dry-run] Kapak indirilecek: %s", meta.ImageURL)
				} else {
					assetID, err := a.uploadCoverImage(ctx, meta.ImageURL, ss.Title)
					if err != nil {
						tui.PrintWarn("Kapak yüklenemedi: %v", err)
					} else {
						patch["coverImage"] = map[string]any{"_type": "image", "asset": ref(assetID)}
					}
				}
			}
		}
	}
	var status string
	if err := tui.SelectOne("Yayın durumu (mevcut: "+ss.UploadStatus+")", append([]string{"(değiştirme)"}, constants.UploadStatuses...), &status); err != nil {
		return err
	}
	if status != "(değiştirme)" {
		patch["uploadStatus"] = status
	}
	var tags []string
	if err := tui.SelectMany("Tür etiketleri (mevcut korunur, boş bırakılabilir)", constants.MangaTags, &tags); err != nil {
		return err
	}
	merged := mergeTags(ss.Tags, tags)
	if !equalTags(merged, ss.Tags) {
		patch["tags"] = merged
	}
	if len(patch) == 0 {
		tui.PrintInfo("Değişiklik yok.")
		return nil
	}
	if a.isDry() {
		tui.PrintInfo("[dry-run] %s yamalanacak (%d alan)", sanityID, len(patch))
		return nil
	}
	if !tui.ConfirmOrAbort(fmt.Sprintf("'%s' güncellensin mi?", ss.Title)) {
		return nil
	}
	before := map[string]any{"title": ss.Title, "description": ss.Description, "tags": ss.Tags, "uploadStatus": ss.UploadStatus}
	if err := a.Client.Patch(ctx, sanityID, patch); err != nil {
		return err
	}
	journal := uploads.NewJournal(sanityID, kind)
	journal.PatchedDocs = append(journal.PatchedDocs, uploads.JournalPatch{ID: sanityID, Rev: ss.Rev, Before: before})
	if err := journal.Save(a.uploadsDir()); err != nil {
		tui.PrintWarn("Günlük kaydedilemedi: %v", err)
	}
	tui.PrintSuccess("Seri güncellendi. Geri almak için 'mangile rollback' kullanın.")
	return nil
}

func (a *App) seriesHasCover(ctx context.Context, id string) (bool, error) {
	var res []map[string]any
	groq := `*[_id == $id && defined(coverImage)][0..0]{_id}`
	if err := a.Client.Query(ctx, groq, map[string]any{"id": id}, &res); err != nil {
		return false, err
	}
	return len(res) > 0, nil
}

func (a *App) uploadCoverImage(ctx context.Context, url, title string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mangile-cli")
	resp, err := coverClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("kapak indirilemedi (%d)", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("kapak boş")
	}
	return a.Client.UploadAsset(ctx, data, formats.Slugify(title)+"-kapak.jpg", "image/jpeg")
}

func askFormat() string {
	var picked string
	opts := append([]string{"(belirtme)"}, constants.MangaFormats...)
	if err := tui.SelectOne("Format", opts, &picked); err != nil {
		return ""
	}
	if picked == "(belirtme)" {
		return ""
	}
	return picked
}

func mergeTags(current, picked []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range current {
		if t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, t := range picked {
		if t != "" && !seen[t] && constants.IsValidTag(t) {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func equalTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
