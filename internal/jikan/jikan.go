package jikan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var BaseURL = "https://api.jikan.moe/v4"

var httpClient = &http.Client{Timeout: 30 * time.Second}

type Meta struct {
	MalID    int
	Title    string
	Synopsis string
	ImageURL string
	Kind     string
}

type mangaResponse struct {
	Data struct {
		MalID    int    `json:"mal_id"`
		Title    string `json:"title"`
		TitleEng string `json:"title_english"`
		Synopsis string `json:"synopsis"`
		Type     string `json:"type"`
		Images   struct {
			JPG struct {
				ImageURL      string `json:"image_url"`
				LargeImageURL string `json:"large_image_url"`
			} `json:"jpg"`
		} `json:"images"`
	} `json:"data"`
}

func FetchManga(malID int) (Meta, error) {
	var meta Meta
	if malID <= 0 {
		return meta, fmt.Errorf("geçersiz MyAnimeList ID: %d", malID)
	}
	url := strings.TrimSuffix(BaseURL, "/") + "/manga/" + strconv.Itoa(malID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return meta, err
	}
	req.Header.Set("User-Agent", "mangile-cli")
	resp, err := httpClient.Do(req)
	if err != nil {
		return meta, fmt.Errorf("Jikan erişilemedi: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return meta, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return meta, fmt.Errorf("MAL ID %d Jikan'da bulunamadı", malID)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return meta, fmt.Errorf("Jikan hız limiti aşıldı, biraz bekleyip tekrar deneyin")
	}
	if resp.StatusCode >= 500 {
		return meta, fmt.Errorf("Jikan/MyAnimeList erişilemiyor (%d), sonra tekrar deneyin", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return meta, fmt.Errorf("Jikan hatası (%d)", resp.StatusCode)
	}
	var parsed mangaResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return meta, fmt.Errorf("Jikan yanıtı okunamadı: %w", err)
	}
	meta.MalID = parsed.Data.MalID
	meta.Title = strings.TrimSpace(parsed.Data.Title)
	if parsed.Data.TitleEng != "" && parsed.Data.TitleEng != parsed.Data.Title {
		meta.Title = strings.TrimSpace(parsed.Data.TitleEng)
	}
	meta.Synopsis = strings.TrimSpace(parsed.Data.Synopsis)
	meta.ImageURL = parsed.Data.Images.JPG.LargeImageURL
	if meta.ImageURL == "" {
		meta.ImageURL = parsed.Data.Images.JPG.ImageURL
	}
	meta.Kind = parsed.Data.Type
	return meta, nil
}
