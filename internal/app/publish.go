package app

import (
	"context"
	"strings"

	"mangile-cli/internal/tui"
)

func (a *App) PublishAll(ctx context.Context) error {
	if err := a.requireToken(); err != nil {
		return err
	}
	tui.PrintTitle("Taslakları Yayınla")
	var ids []string
	groq := `*[_id match "drafts.**"]._id`
	if err := a.Client.Query(ctx, groq, nil, &ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		tui.PrintInfo("Yayınlanacak taslak yok.")
		return nil
	}
	clean := make([]string, 0, len(ids))
	for _, id := range ids {
		if strings.HasPrefix(id, "drafts.") {
			clean = append(clean, id)
		}
	}
	tui.PrintInfo("Bulunan taslak: %d", len(clean))
	if !tui.ConfirmOrAbort("Tüm taslaklar yayınlansın mı?") {
		return nil
	}
	if a.isDry() {
		tui.PrintInfo("  [dry-run] %d taslak yayınlanacak", len(clean))
		return nil
	}
	if err := a.Client.Publish(ctx, clean); err != nil {
		return err
	}
	tui.PrintSuccess("%d taslak yayınlandı.", len(clean))
	return nil
}
