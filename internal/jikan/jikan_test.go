package jikan

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchManga(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manga/3" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"mal_id":3,"title":"20-seiki Shounen","title_english":"20th Century Boys","synopsis":"Bir kehanet.","type":"Manga","images":{"jpg":{"image_url":"https://k/k.jpg","large_image_url":"https://k/kb.jpg"}}}}`))
	}))
	defer srv.Close()
	old := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = old }()

	meta, err := FetchManga(3)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "20th Century Boys" {
		t.Errorf("İngilizce başlık öncelikli olmalı: %q", meta.Title)
	}
	if meta.ImageURL != "https://k/kb.jpg" {
		t.Errorf("büyük görsel öncelikli olmalı: %q", meta.ImageURL)
	}
	if meta.Synopsis != "Bir kehanet." || meta.Kind != "Manga" {
		t.Errorf("alanlar eksik: %+v", meta)
	}
}

func TestFetchMangaNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	old := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = old }()

	if _, err := FetchManga(999999); err == nil {
		t.Fatal("bulunamayan ID hata vermeli")
	}
	if _, err := FetchManga(0); err == nil {
		t.Fatal("geçersiz ID hata vermeli")
	}
}
