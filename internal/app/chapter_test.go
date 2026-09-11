package app

import (
	"strings"
	"testing"
)

func TestChapterDisplay(t *testing.T) {
	c := sanityChapter{ID: "mangaChapter-3-0-12", Type: "mangaChapter", Title: "Başlangıç", Number: 12}
	if got := c.display(); !strings.Contains(got, "Bölüm 12") || !strings.Contains(got, "Başlangıç") {
		t.Errorf("beklenmeyen etiket: %q", got)
	}
	draft := sanityChapter{ID: "drafts.mangaChapter-3-0-12", Number: 12}
	if got := draft.display(); !strings.Contains(got, "taslak") {
		t.Errorf("taslak işareti yok: %q", got)
	}
	vol := sanityChapter{ID: "x", Number: 5, Volume: 2}
	if got := vol.display(); !strings.Contains(got, "Cilt 2") {
		t.Errorf("cilt öneki yok: %q", got)
	}
}

func TestChapterTwinIDs(t *testing.T) {
	pub := chapterTwinIDs("mangaChapter-3-0-12")
	if len(pub) != 2 || pub[1] != "drafts.mangaChapter-3-0-12" {
		t.Errorf("yayınlı ikizi yanlış: %v", pub)
	}
	draft := chapterTwinIDs("drafts.mangaChapter-3-0-12")
	if len(draft) != 2 || draft[1] != "mangaChapter-3-0-12" {
		t.Errorf("taslak ikizi yanlış: %v", draft)
	}
}

func TestChapterActionLabel(t *testing.T) {
	cases := map[string]string{"list": "listele", "edit": "düzenle", "delete": "sil", "": "seç", "x": "seç"}
	for in, want := range cases {
		if got := chapterActionLabel(in); got != want {
			t.Errorf("label(%q) = %q, beklenen %q", in, got, want)
		}
	}
}
