package app

import (
	"context"
	"fmt"
	"strings"

	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

func (a *App) Rollback(ctx context.Context) error {
	if err := a.requireToken(); err != nil {
		return err
	}
	tui.PrintTitle("Geri Alma (Rollback)")
	journals, err := uploads.ListJournals(a.uploadsDir())
	if err != nil {
		return err
	}
	if len(journals) == 0 {
		tui.PrintInfo("Geri alınacak işlem günlüğü yok.")
		return nil
	}
	var labels []string
	for _, j := range journals {
		labels = append(labels, fmt.Sprintf("%s | %s | %d bölüm, %d asset", j.ID, j.CreatedAt.Format("2006-01-02 15:04"), len(j.CreatedDocs), len(j.Assets)))
	}
	var pick string
	if err := tui.SelectOne("Geri alınacak işlem", labels, &pick); err != nil {
		return err
	}
	var journal *uploads.Journal
	for i, l := range labels {
		if l == pick {
			journal = journals[i]
			break
		}
	}
	if journal == nil {
		return fmt.Errorf("geçersiz seçim")
	}
	if !tui.ConfirmOrAbort(fmt.Sprintf("'%s' işlemi tamamen geri alınsın mı?", journal.ID)) {
		return nil
	}

	var allIDs []string
	for _, id := range journal.CreatedDocs {
		allIDs = append(allIDs, id)
		allIDs = append(allIDs, strings.TrimPrefix(id, "drafts."))
	}
	existing, err := a.filterExisting(ctx, allIDs)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		if err := a.Client.Delete(ctx, existing); err != nil {
			return err
		}
		tui.PrintInfo("Silinen bölüm doc'ları: %d", len(existing))
	}

	for _, p := range journal.PatchedDocs {
		if err := a.revertPatch(ctx, p.ID, p.Rev, p.Before); err != nil {
			tui.PrintWarn("Patch geri alınamadı (%s): %v", p.ID, err)
		} else {
			tui.PrintInfo("Patch geri alındı: %s", p.ID)
		}
	}

	removed := a.deleteOrphanAssets(ctx, journal.Assets)
	tui.PrintSuccess("Geri alma tamamlandı. Silinen bölüm: %d, temizlenen asset: %d", len(existing), removed)
	_ = uploads.DeleteJournal(a.uploadsDir(), journal.ID)
	return nil
}

func (a *App) filterExisting(ctx context.Context, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var existing []string
	groq := `*[_id in $ids]._id`
	if err := a.Client.Query(ctx, groq, map[string]any{"ids": ids}, &existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (a *App) revertPatch(ctx context.Context, id, rev string, before map[string]any) error {
	if rev != "" {
		if err := a.Client.Revert(ctx, id, rev); err == nil {
			return nil
		}
	}
	if len(before) > 0 {
		return a.Client.Patch(ctx, id, before)
	}
	return fmt.Errorf("geri dönüş yedeği yok: %s", id)
}

func (a *App) deleteOrphanAssets(ctx context.Context, assetIDs []string) int {
	var deleted int
	for _, assetID := range assetIDs {
		refs, err := a.assetReferenceCount(ctx, assetID)
		if err != nil {
			continue
		}
		if refs == 0 {
			if err := a.Client.Delete(ctx, []string{assetID}); err == nil {
				deleted++
			}
		}
	}
	return deleted
}

func (a *App) assetReferenceCount(ctx context.Context, assetID string) (int, error) {
	var count []int
	groq := `count(*[references($assetId)])`
	params := map[string]any{"assetId": assetID}
	if err := a.Client.Query(ctx, groq, params, &count); err != nil {
		return 0, err
	}
	if len(count) == 0 {
		return 0, nil
	}
	return count[0], nil
}
