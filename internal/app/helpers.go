package app

import (
	"context"
	"fmt"

	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

func (a *App) resolveSeriesID(ctx context.Context, series *uploads.Series, expectedType string) (string, error) {
	if series.Config.SanityID != "" {
		ss, err := a.fetchSeriesByID(ctx, series.Config.SanityID)
		if err != nil {
			return "", err
		}
		if ss != nil {
			if ss.Type != expectedType {
				return "", fmt.Errorf("%s Sanity'de %s tipinde değil (%s)", ss.ID, expectedType, ss.Type)
			}
			return ss.ID, nil
		}
		tui.PrintWarn("config.yaml'deki sanityId bulunamadı, MAL ID ile aranıyor…")
	}
	if series.Config.MalID <= 0 {
		return "", fmt.Errorf("config.yaml'de myAnimeListId tanımlı değil: %s/config.yaml", series.Dir)
	}
	ss, err := a.findSanitySeries(ctx, series.Config.MalID, expectedType)
	if err != nil {
		return "", err
	}
	if ss == nil {
		return "", fmt.Errorf("Sanity'de MAL ID %d ile %s bulunamadı. Önce Sanity'de seri oluşturun.", series.Config.MalID, expectedType)
	}
	series.Config.SanityID = ss.ID
	series.Config.Type = expectedType
	if err := uploads.SaveSeriesConfig(series.Dir, series.Config); err != nil {
		return "", err
	}
	tui.PrintInfo("Sanity eşleşmesi: %s", ss.ID)
	return ss.ID, nil
}

type scanDoc struct {
	ID     string `json:"_id"`
	Rev    string `json:"_rev"`
	Name   string `json:"name"`
	Titles []struct {
		Ref string `json:"_ref"`
	} `json:"titles"`
}

func (a *App) syncScanTitle(ctx context.Context, journal *uploads.Journal, scanID, seriesID string, created bool) error {
	if seriesID == "" {
		return nil
	}
	var docs []scanDoc
	groq := `*[_type == "scan" && _id == $id][0..0]{_id, _rev, name, titles}`
	if err := a.Client.Query(ctx, groq, map[string]any{"id": scanID}, &docs); err != nil {
		return err
	}
	if len(docs) == 0 {
		tui.PrintWarn("Scan bulunamadı: %s (titles[] senkronu atlandı)", scanID)
		return nil
	}
	doc := docs[0]
	for _, t := range doc.Titles {
		if t.Ref == seriesID {
			return nil
		}
	}
	before := map[string]any{}
	if doc.Titles != nil {
		refs := make([]map[string]any, 0, len(doc.Titles))
		for _, t := range doc.Titles {
			refs = append(refs, map[string]any{"_type": "reference", "_ref": t.Ref})
		}
		before["titles"] = refs
	}
	if err := a.Client.Patch(ctx, scanID, map[string]any{
		"titles": append(beforeRefs(before), ref(seriesID)),
	}); err != nil {
		return err
	}
	_ = created
	journal.PatchedDocs = append(journal.PatchedDocs, uploads.JournalPatch{ID: scanID, Rev: doc.Rev, Before: before})
	return nil
}

func beforeRefs(before map[string]any) []map[string]any {
	if v, ok := before["titles"]; ok {
		if refs, ok := v.([]map[string]any); ok {
			return refs
		}
	}
	return []map[string]any{}
}

func (a *App) publishIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if a.isDry() {
		tui.PrintInfo("  [dry-run] %d taslak yayınlanacak", len(ids))
		return nil
	}
	return a.Client.Publish(ctx, ids)
}
