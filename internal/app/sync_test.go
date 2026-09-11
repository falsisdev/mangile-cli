package app

import (
	"testing"
)

func TestSyncPlan(t *testing.T) {
	chapters := []sanityChapter{
		{ID: "drafts.mangaChapter-3-0-1", Number: 1},
		{ID: "mangaChapter-3-0-1", Number: 1},
		{ID: "mangaChapter-3-0-2", Number: 2},
	}
	local := map[string]bool{"2": true}
	pending, skipped := syncPlan(chapters, local)
	if skipped != 1 {
		t.Errorf("1 atlama beklenir: %d", skipped)
	}
	if len(pending) != 1 {
		t.Fatalf("1 indirme beklenir: %d", len(pending))
	}
	if pending[0].ID != "mangaChapter-3-0-1" {
		t.Errorf("yayınlı kopya öncelikli olmalı: %s", pending[0].ID)
	}
}

func TestSyncPlanAllLocal(t *testing.T) {
	chapters := []sanityChapter{{ID: "mangaChapter-3-0-5", Number: 5}}
	pending, skipped := syncPlan(chapters, map[string]bool{"5": true})
	if len(pending) != 0 || skipped != 1 {
		t.Errorf("hepsi yerelde olmalı: %d/%d", len(pending), skipped)
	}
}

func TestNovelDataLine(t *testing.T) {
	if got := novelDataLine(2, 12); got != "Cilt 2 Bölüm 12" {
		t.Errorf("yanlış satır: %q", got)
	}
	if got := novelDataLine(0, 0); got != "Bölüm 0" {
		t.Errorf("ciltsiz satır yanlış: %q", got)
	}
}

func TestImageExtFromURL(t *testing.T) {
	cases := map[string]string{
		"https://cdn.sanity.io/a.png":       ".png",
		"https://cdn.sanity.io/a.JPG?w=100": ".jpg",
		"https://cdn.sanity.io/uzantisiz":   ".jpg",
	}
	for in, want := range cases {
		if got := imageExtFromURL(in); got != want {
			t.Errorf("ext(%s) = %s, beklenen %s", in, got, want)
		}
	}
}
