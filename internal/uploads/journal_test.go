package uploads

import (
	"testing"
)

func TestJournalRoundTrip(t *testing.T) {
	uploads := t.TempDir()
	j := NewJournal("seri-1", "manga")
	j.CreatedDocs = append(j.CreatedDocs, "drafts.a", "drafts.b")
	j.Assets = append(j.Assets, "image-a")
	j.PatchedDocs = append(j.PatchedDocs, JournalPatch{ID: "scan-1", Rev: "r1", Before: map[string]any{"titles": []any{}}})
	if err := j.Save(uploads); err != nil {
		t.Fatal(err)
	}
	got, err := LoadJournal(uploads, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != j.ID || len(got.CreatedDocs) != 2 || got.Assets[0] != "image-a" ||
		len(got.PatchedDocs) != 1 || got.PatchedDocs[0].Rev != "r1" {
		t.Errorf("journal roundtrip farklı: %+v", got)
	}
}

func TestListDeleteJournals(t *testing.T) {
	uploads := t.TempDir()
	j1 := NewJournal("a", "manga")
	j1.ID = "20260101-120000"
	if err := j1.Save(uploads); err != nil {
		t.Fatal(err)
	}
	j2 := NewJournal("b", "lightNovel")
	j2.ID = "20260102-120000"
	if err := j2.Save(uploads); err != nil {
		t.Fatal(err)
	}
	list, err := ListJournals(uploads)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("2 journal beklenir, %d", len(list))
	}
	if err := DeleteJournal(uploads, j1.ID); err != nil {
		t.Fatal(err)
	}
	list, _ = ListJournals(uploads)
	if len(list) != 1 {
		t.Fatalf("1 journal beklenir, %d", len(list))
	}
}
