package app

import (
	"os"
	"path/filepath"
	"testing"

	"mangile-cli/internal/constants"
)

func TestPlanCSVFolders(t *testing.T) {
	headers := []string{"data", "baslik", "text"}
	rows := [][]string{
		{"Cilt 2 Bölüm 12", "Başlangıç", "metin"},
		{"Cilt 2 Bölüm 13", "Devam: İkinci Kısım", "metin2"},
	}
	plan, err := planCSVFolders(headers, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 {
		t.Fatalf("2 klasör beklenir: %d", len(plan))
	}
	if plan[0].folder != "Cilt 002 Bölüm 012 - Başlangıç" {
		t.Errorf("klasör adı yanlış: %q", plan[0].folder)
	}
	if plan[1].folder != "Cilt 002 Bölüm 013 - Devam - İkinci Kısım" {
		t.Errorf("iki nokta dönüşümü yanlış: %q", plan[1].folder)
	}
	if plan[0].files["text.txt"] != "metin" {
		t.Error("sütun dosyası yazılmalı")
	}
}

func TestPlanCSVFoldersMissingColumns(t *testing.T) {
	if _, err := planCSVFolders([]string{"x", "y"}, [][]string{{"a", "b"}}); err == nil {
		t.Fatal("eksik sütun hata vermeli")
	}
}

func TestPlanCSVFoldersDuplicate(t *testing.T) {
	headers := []string{"data", "baslik"}
	rows := [][]string{{"Bölüm 1", "Aynı"}, {"Bölüm 1", "Aynı"}}
	plan, err := planCSVFolders(headers, rows)
	if err != nil {
		t.Fatal(err)
	}
	if plan[0].folder == plan[1].folder {
		t.Error("yinelenen klasör adı benzersizleşmeli")
	}
}

func TestReadCSVRows(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "veri.csv")
	content := "data,baslik,text\nCilt 1 Bölüm 1,Merhaba,dünya\n"
	if err := os.WriteFile(csvPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, headers, err := readCSVRows(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(headers) != 3 || len(rows) != 1 || rows[0][1] != "Merhaba" {
		t.Errorf("CSV okuma yanlış: %v %v", headers, rows)
	}
}

func TestParseMigrateTitle(t *testing.T) {
	vol, num, clean := parseMigrateTitle("Cilt 2 Bölüm 12: Başlangıç")
	if vol != 2 || num != 12 || clean != "Başlangıç" {
		t.Errorf("yanlış ayrıştırma: %d %d %q", vol, num, clean)
	}
	_, num, _ = parseMigrateTitle("Önsöz")
	if num != constants.NovelChapterPrologue {
		t.Errorf("önsöz 0 olmalı: %d", num)
	}
	_, num, _ = parseMigrateTitle("Sonsöz")
	if num != constants.NovelChapterEpilogue {
		t.Errorf("sonsöz 999 olmalı: %d", num)
	}
	_, num, _ = parseMigrateTitle("Yan Hikaye: Ekstra")
	if num != constants.NovelChapterExtra {
		t.Errorf("ekstra 1000 olmalı: %d", num)
	}
}

func TestSanitizeCSVName(t *testing.T) {
	if got := sanitizeCSVName("A:B/C"); got != "A -B_C" {
		t.Errorf("temizleme yanlış: %q", got)
	}
}
