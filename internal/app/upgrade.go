package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"mangile-cli/internal/formats"
	"mangile-cli/internal/tui"
)

var releaseAPIURL = "https://api.github.com/repos/falsisdev/mangile-cli/releases/latest"

var upgradeClient = &http.Client{Timeout: 120 * time.Second}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type releaseInfo struct {
	Tag    string         `json:"tag_name"`
	Assets []releaseAsset `json:"assets"`
}

func (a *App) Upgrade(ctx context.Context) error {
	tui.PrintTitle("Sürüm Yükseltme")
	current := strings.TrimSpace(a.Cfg.Version)
	if current == "" {
		current = "dev"
	}
	tui.PrintInfo("Kurulu sürüm: %s", current)
	if a.isDry() {
		tui.PrintInfo("[dry-run] En son sürüm denetlenip çalıştırılabilir dosya değiştirilecek: %s", exePath())
		return nil
	}
	rel, err := fetchLatestRelease(ctx)
	if err != nil {
		return err
	}
	tui.PrintInfo("En son sürüm: %s", rel.Tag)
	if !isNewerVersion(current, rel.Tag) {
		tui.PrintSuccess("Zaten güncel sürümdesiniz.")
		return nil
	}
	assetName := upgradeAssetName(rel.Tag, runtime.GOOS, runtime.GOARCH)
	url := ""
	for _, asset := range rel.Assets {
		if asset.Name == assetName {
			url = asset.URL
			break
		}
	}
	if url == "" {
		return fmt.Errorf("bu platforma uygun dosya bulunamadı (%s)", assetName)
	}
	if !tui.ConfirmOrAbort(fmt.Sprintf("%s sürümüne yükseltilsin mi?", rel.Tag)) {
		return nil
	}
	target := exePath()
	tui.PrintInfo("İndiriliyor: %s", assetName)
	data, err := downloadURL(ctx, url, 64<<20)
	if err != nil {
		return err
	}
	if err := verifyChecksum(ctx, rel.Assets, assetName, data); err != nil {
		return err
	}
	bin, err := extractBinary(data, assetName)
	if err != nil {
		return err
	}
	if err := installBinary(bin, target); err != nil {
		return err
	}
	tui.PrintSuccess("Yükseltme tamamlandı (%s). Yeni sürümü görmek için terminali yeniden başlatın.", rel.Tag)
	return nil
}

func fetchLatestRelease(ctx context.Context) (releaseInfo, error) {
	var rel releaseInfo
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPIURL, nil)
	if err != nil {
		return rel, err
	}
	req.Header.Set("User-Agent", "mangile-cli")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := upgradeClient.Do(req)
	if err != nil {
		return rel, fmt.Errorf("sürüm bilgisi alınamadı: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return rel, fmt.Errorf("sürüm bilgisi alınamadı (%d)", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return rel, err
	}
	if err := json.Unmarshal(data, &rel); err != nil {
		return rel, fmt.Errorf("sürüm yanıtı okunamadı: %w", err)
	}
	if rel.Tag == "" {
		return rel, fmt.Errorf("sürüm yanıtı boş")
	}
	return rel, nil
}

func upgradeAssetName(tag, goos, goarch string) string {
	ver := strings.TrimPrefix(strings.TrimSpace(tag), "v")
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return "mangile_" + ver + "_" + goos + "_" + goarch + "." + ext
}

func isNewerVersion(current, latest string) bool {
	cur := strings.TrimPrefix(strings.TrimSpace(current), "v")
	lat := strings.TrimPrefix(strings.TrimSpace(latest), "v")
	if cur == "" || cur == "dev" {
		return true
	}
	return cur != lat
}

func verifyChecksum(ctx context.Context, assets []releaseAsset, assetName string, data []byte) error {
	url := ""
	for _, asset := range assets {
		if asset.Name == "checksums.txt" {
			url = asset.URL
			break
		}
	}
	if url == "" {
		tui.PrintWarn("checksums.txt bulunamadı, doğrulama atlandı.")
		return nil
	}
	raw, err := downloadURL(ctx, url, 1<<20)
	if err != nil {
		return fmt.Errorf("sağlama dosyası indirilemedi: %w", err)
	}
	want := ""
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("sağlama dosyasında kayıt yok: %s", assetName)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != strings.ToLower(want) {
		return fmt.Errorf("sağlama tutmadı, indirme bozuk olabilir")
	}
	return nil
}

func extractBinary(data []byte, assetName string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "mangile-upgrade-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	archive := filepath.Join(dir, assetName)
	if err := os.WriteFile(archive, data, 0o600); err != nil {
		return nil, err
	}
	out := filepath.Join(dir, "out")
	if err := formats.ExpandArchive(archive, out); err != nil {
		return nil, fmt.Errorf("arşiv açılamadı: %w", err)
	}
	for _, name := range []string{"mangile", "mangile.exe"} {
		path := filepath.Join(out, name)
		if bin, err := os.ReadFile(path); err == nil && len(bin) > 0 {
			return bin, nil
		}
	}
	return nil, fmt.Errorf("arşivde çalıştırılabilir dosya yok")
}

func installBinary(bin []byte, target string) error {
	if target == "" {
		return fmt.Errorf("hedef yol boş")
	}
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, "mangile-new-")
	if err != nil {
		return fmt.Errorf("yazma izni yok (%s): %w", dir, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		os.Remove(tmpName)
		return err
	}
	backup := target + ".old"
	os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("mevcut dosya yedeklenemedi: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Rename(backup, target)
		return fmt.Errorf("yeni dosya taşınamadı, eski sürüm geri alındı: %w", err)
	}
	os.Remove(backup)
	return nil
}

func exePath() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return path
}
