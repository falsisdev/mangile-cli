package app

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUpgradeAssetName(t *testing.T) {
	cases := map[[3]string]string{
		{"v0.2.0", "darwin", "arm64"}:  "mangile_0.2.0_darwin_arm64.tar.gz",
		{"0.2.0", "linux", "amd64"}:    "mangile_0.2.0_linux_amd64.tar.gz",
		{"v0.2.0", "windows", "amd64"}: "mangile_0.2.0_windows_amd64.zip",
	}
	for in, want := range cases {
		if got := upgradeAssetName(in[0], in[1], in[2]); got != want {
			t.Errorf("asset(%v) = %s, beklenen %s", in, got, want)
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	if !isNewerVersion("v0.1.0", "v0.2.0") {
		t.Error("yeni sürüm algılanmalı")
	}
	if isNewerVersion("v0.2.0", "v0.2.0") {
		t.Error("aynı sürüm güncel sayılmalı")
	}
	if isNewerVersion("0.2.0", "v0.2.0") {
		t.Error("v öneki farkı gözetilmemeli")
	}
	if !isNewerVersion("dev", "v0.2.0") {
		t.Error("dev sürümü yükseltilebilir olmalı")
	}
}

func TestInstallBinary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "mangile")
	if err := os.WriteFile(target, []byte("eski"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := installBinary([]byte("yeni"), target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "yeni" {
		t.Errorf("dosya değişmeli: %q", got)
	}
	if _, err := os.Stat(target + ".old"); !os.IsNotExist(err) {
		t.Error("yedek temizlenmeli")
	}
}

func TestInstallBinaryNoTarget(t *testing.T) {
	if err := installBinary([]byte("x"), ""); err == nil {
		t.Error("boş hedef hata vermeli")
	}
}

func makeTestZip(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractBinary(t *testing.T) {
	data := makeTestZip(t, "mangile", "sahte-binary")
	bin, err := extractBinary(data, "mangile_0.9.9_linux_amd64.zip")
	if err != nil {
		t.Fatal(err)
	}
	if string(bin) != "sahte-binary" {
		t.Errorf("yanlış içerik: %q", bin)
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := makeTestZip(t, "mangile", "sahte-binary")
	sum := sha256.Sum256(data)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(hex.EncodeToString(sum[:]) + "  mangile_0.9.9_linux_amd64.zip\n"))
	}))
	defer srv.Close()
	assets := []releaseAsset{{Name: "checksums.txt", URL: srv.URL + "/checksums.txt"}}
	if err := verifyChecksum(context.Background(), assets, "mangile_0.9.9_linux_amd64.zip", data); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(context.Background(), assets, "mangile_0.9.9_linux_amd64.zip", []byte("bozuk")); err == nil {
		t.Error("bozuk veri hata vermeli")
	}
}
