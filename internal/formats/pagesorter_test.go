package formats

import "testing"

func TestDetectIndex(t *testing.T) {
	tests := []struct {
		name string
		want int
		tier int
		ok   bool
	}{
		{"001.png", 1, 1, true},
		{"1.png", 1, 1, true},
		{"0123.jpg", 123, 1, true},
		{"page12.jpg", 12, 2, true},
		{"page_03.webp", 3, 2, true},
		{"sayfa-07.png", 7, 2, true},
		{"ch56-7.png", 7, 3, true},
		{"1_001.jpg", 1, 3, true},
		{"409-1.png", 1, 3, true},
		{"cover.png", 0, 0, false},
		{"scan02.jpg", 2, 2, true},
		{"hello.png", 0, 0, false},
	}
	for _, tt := range tests {
		got, tier, ok := DetectIndex(tt.name)
		if got != tt.want || tier != tt.tier || ok != tt.ok {
			t.Errorf("DetectIndex(%q) = (%d, %d, %v), want (%d, %d, %v)", tt.name, got, tier, ok, tt.want, tt.tier, tt.ok)
		}
	}
}

func TestSortPagesNumeric(t *testing.T) {
	entries := []Entry{
		{Name: "10.png"},
		{Name: "2.png"},
		{Name: "1.png"},
	}
	res := SortPages(entries)
	want := []string{"1.png", "2.png", "10.png"}
	for i, w := range want {
		if res.Entries[i].Name != w {
			t.Fatalf("sıra %d = %s, want %s", i, res.Entries[i].Name, w)
		}
	}
}

func TestSortPagesPaddingEquality(t *testing.T) {
	entries := []Entry{
		{Name: "001.png"},
		{Name: "01.png"},
		{Name: "00003.png"},
		{Name: "4.png"},
	}
	res := SortPages(entries)
	want := []string{"001.png", "01.png", "00003.png", "4.png"}
	if len(res.Warnings) == 0 {
		t.Errorf("duplike indeks uyarısı bekleniyor, alınmadı")
	}
	for i, w := range want {
		if res.Entries[i].Name != w {
			t.Fatalf("sıra %d = %s, want %s", i, res.Entries[i].Name, w)
		}
	}
}

func TestSortPagesUnnumbered(t *testing.T) {
	entries := []Entry{
		{Name: "arka.png"},
		{Name: "cover.png"},
		{Name: "2.png"},
	}
	res := SortPages(entries)
	if res.Entries[0].Name != "2.png" {
		t.Fatalf("ilk sıra %s olmalı (numaralılar önce)", res.Entries[0].Name)
	}
	hasWarn := false
	for _, w := range res.Warnings {
		if w == "numara içermiyor, ada göre dizildi: cover.png" || w == "numara içermiyor, ada göre dizildi: arka.png" {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Errorf("numarasız dosya uyarısı bekleniyor")
	}
}

func TestSortPagesPrefixed(t *testing.T) {
	entries := []Entry{
		{Name: "page12.jpg"},
		{Name: "page2.jpg"},
		{Name: "page1.jpg"},
	}
	res := SortPages(entries)
	want := []string{"page1.jpg", "page2.jpg", "page12.jpg"}
	for i, w := range want {
		if res.Entries[i].Name != w {
			t.Fatalf("sıra %d = %s, want %s", i, res.Entries[i].Name, w)
		}
	}
}

func TestNaturalLess(t *testing.T) {
	cases := [][2]string{
		{"1", "2"},
		{"2", "10"},
		{"1.png", "1_2.png"},
		{"a1", "a10"},
		{"a2b", "a10b"},
		{"ch56-1", "ch56-10"},
	}
	for _, c := range cases {
		if !NaturalLess(c[0], c[1]) {
			t.Errorf("NaturalLess(%q, %q) = false, want true", c[0], c[1])
		}
		if NaturalLess(c[1], c[0]) {
			t.Errorf("NaturalLess(%q, %q) = true, want false", c[1], c[0])
		}
	}
}
