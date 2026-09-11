package app

import (
	"testing"
)

func TestParseChapterNumber(t *testing.T) {
	cases := map[string]struct {
		want float64
		ok   bool
	}{
		"Bölüm 12":          {12, true},
		"Chapter 3.5":       {3.5, true},
		"Bölüm 45 - Başlık": {45, true},
		"Cilt 2 Bölüm 10":   {10, true},
		"123":               {123, true},
		"Önsöz":             {0, false},
		"prologue":          {0, false},
	}
	for in, c := range cases {
		got, ok := parseChapterNumber(in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseChapterNumber(%q) = %v,%v istediğim %v,%v", in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseVolume(t *testing.T) {
	cases := map[string]struct {
		want int
		ok   bool
	}{
		"Cilt 12":      {12, true},
		"Cilt 2 Bölüm": {2, true},
		"Bölüm 5":      {0, false},
		"Cilt 0":       {0, true},
	}
	for in, c := range cases {
		got, ok := parseVolume(in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseVolume(%q) = %d,%v istediğim %d,%v", in, got, ok, c.want, c.ok)
		}
	}
}

func TestCleanChapterTitle(t *testing.T) {
	cases := map[string]string{
		"Bölüm 45 - Son Bahar": "Son Bahar",
		"Bölüm 45: Başlık":     "Başlık",
		"Chapter 3 - Test":     "Test",
		"Bölüm 5":              "",
		"chapter 7:Gizem":      "Gizem",
	}
	for in, want := range cases {
		if got := cleanChapterTitle(in); got != want {
			t.Errorf("cleanChapterTitle(%q) = %q, istediğim %q", in, got, want)
		}
	}
}

func TestChapterIDDeterministic(t *testing.T) {
	a := chapterID("mangaChapter", 12345, 3, "12")
	b := chapterID("mangaChapter", 12345, 3, "12.0")
	c := chapterID("mangaChapter", 12345, 0, "12")
	d := chapterID("novelChapter", 12345, 0, "12")
	if a != "mangaChapter-12345-3-12" {
		t.Errorf("beklenen mangaChapter-12345-3-12, %s", a)
	}
	if a != b {
		t.Errorf("12 ve 12.0 aynı olmalı: %s vs %s", a, b)
	}
	if c != "mangaChapter-12345-0-12" {
		t.Errorf("beklenen mangaChapter-12345-0-12, %s", c)
	}
	if d != "novelChapter-12345-0-12" {
		t.Errorf("beklenen novelChapter-12345-0-12, %s", d)
	}
}

func TestFormatNum(t *testing.T) {
	cases := map[float64]string{
		5:    "5",
		3.5:  "3.5",
		12:   "12",
		0:    "0",
		10.0: "10",
	}
	for f, want := range cases {
		if got := formatNum(f); got != want {
			t.Errorf("formatNum(%v) = %q, istediğim %q", f, got, want)
		}
	}
}
