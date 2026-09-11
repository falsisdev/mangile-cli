package app

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mangile-cli/internal/cfg"
	"mangile-cli/internal/uploads"
)

func newWebTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	if err := uploads.SaveSeriesConfig(filepath.Join(dir, "Deneme"), uploads.SeriesConfig{MalID: 3, Type: "manga", Title: "Deneme"}); err != nil {
		t.Fatal(err)
	}
	c := cfg.Config{ProjectID: "p", Dataset: "d", APIVersion: "v", UploadsDir: dir, WebPort: 8787}
	return New(c)
}

func TestWebIndexListsSeries(t *testing.T) {
	a := newWebTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	a.webMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("durum %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Deneme") {
		t.Error("seri listede görünmeli")
	}
	if !strings.Contains(body, "/upload") {
		t.Error("form /upload adresine gitmeli")
	}
}

func postUpload(t *testing.T, a *App, series, chapter string, files map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("series", series)
	_ = mw.WriteField("chapter", chapter)
	for name, content := range files {
		fw, err := mw.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	a.webMux().ServeHTTP(rec, req)
	return rec
}

func TestWebUploadSavesImages(t *testing.T) {
	a := newWebTestApp(t)
	rec := postUpload(t, a, "Deneme", "Bölüm 1", map[string]string{"001.jpg": "jpegveri", "002.png": "pngveri"})
	if rec.Code != http.StatusOK {
		t.Fatalf("durum %d: %s", rec.Code, rec.Body.String())
	}
	for _, name := range []string{"001.jpg", "002.png"} {
		if _, err := os.Stat(filepath.Join(a.uploadsDir(), "Deneme", "Bölüm 1", name)); err != nil {
			t.Errorf("dosya yazılmalı: %s", name)
		}
	}
}

func TestWebUploadRejectsNonImages(t *testing.T) {
	a := newWebTestApp(t)
	rec := postUpload(t, a, "Deneme", "Bölüm 2", map[string]string{"not.txt": "metin", "kotu.exe": "calistirilabilir"})
	if rec.Code == http.StatusOK {
		t.Fatal("görsel olmayan dosyalar kabul edilmemeli")
	}
	if _, err := os.Stat(filepath.Join(a.uploadsDir(), "Deneme", "Bölüm 2", "not.txt")); !os.IsNotExist(err) {
		t.Error("txt dosyası yazılmamalı")
	}
}

func TestWebUploadRejectsTraversal(t *testing.T) {
	a := newWebTestApp(t)
	rec := postUpload(t, a, "Deneme", "../kacis", map[string]string{"001.jpg": "x"})
	if rec.Code == http.StatusOK {
		t.Fatal("yol kaçışı kabul edilmemeli")
	}
	if _, err := os.Stat(filepath.Join(a.uploadsDir(), "kacis")); !os.IsNotExist(err) {
		t.Error("üst dizine yazılmamalı")
	}
}

func TestWebUploadUnknownSeries(t *testing.T) {
	a := newWebTestApp(t)
	rec := postUpload(t, a, "Olmayan", "Bölüm 1", map[string]string{"001.jpg": "x"})
	if rec.Code == http.StatusOK {
		t.Fatal("bilinmeyen seri kabul edilmemeli")
	}
}

func TestWebUploadUniqueNames(t *testing.T) {
	a := newWebTestApp(t)
	first := postUpload(t, a, "Deneme", "Bölüm 3", map[string]string{"001.jpg": "bir"})
	if first.Code != http.StatusOK {
		t.Fatalf("ilk yükleme: %d", first.Code)
	}
	second := postUpload(t, a, "Deneme", "Bölüm 3", map[string]string{"001.jpg": "iki"})
	if second.Code != http.StatusOK {
		t.Fatalf("ikinci yükleme: %d", second.Code)
	}
	if _, err := os.Stat(filepath.Join(a.uploadsDir(), "Deneme", "Bölüm 3", "001-2.jpg")); err != nil {
		t.Error("aynı isim benzersizleşmeli: 001-2.jpg")
	}
}

func TestWebSafeImageName(t *testing.T) {
	if _, ok := webSafeImageName("sayfa.PNG"); !ok {
		t.Error("png kabul edilmeli (büyük harf dahil)")
	}
	if _, ok := webSafeImageName("belge.pdf"); ok {
		t.Error("pdf reddedilmeli")
	}
	if _, ok := webSafeImageName(".gizli.jpg"); ok {
		t.Error("nokta ile başlayan ad reddedilmeli")
	}
	if got, ok := webSafeImageName("a/b.jpg"); !ok || got != "b.jpg" {
		t.Errorf("yol temizlenmeli, %q", got)
	}
}
