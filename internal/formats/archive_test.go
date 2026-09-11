package formats

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func writeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.zip")
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeTarGz(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.tar.gz")
	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writePlainTar(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.tar")
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExpandZip(t *testing.T) {
	path := writeZip(t, map[string]string{
		"001.png":       "png1",
		"sub/002.png":   "png2",
		"ComicInfo.xml": "<ComicInfo/>",
	})
	dst := t.TempDir()
	if err := ExpandArchive(path, dst); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"001.png", filepath.Join("sub", "002.png"), "ComicInfo.xml"} {
		if _, err := os.Stat(filepath.Join(dst, f)); err != nil {
			t.Errorf("eksik dosya %s: %v", f, err)
		}
	}
}

func TestExpandTarGz(t *testing.T) {
	path := writeTarGz(t, map[string]string{
		"001.png":     "png1",
		"sub/002.png": "png2",
	})
	dst := t.TempDir()
	if err := ExpandArchive(path, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "sub", "002.png")); err != nil {
		t.Errorf("eksik dosya: %v", err)
	}
}

func TestExpandPlainTar(t *testing.T) {
	path := writePlainTar(t, map[string]string{
		"001.png":     "png1",
		"sub/002.png": "png2",
	})
	dst := t.TempDir()
	if err := ExpandArchive(path, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "sub", "002.png")); err != nil {
		t.Errorf("eksik dosya: %v", err)
	}
}

func TestExpandArchiveUnsupported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.bin")
	_ = os.WriteFile(path, []byte{0x1f, 0x9d, 0x00, 0x00}, 0o644)
	if err := ExpandArchive(path, t.TempDir()); err == nil {
		t.Fatal("desteklenmeyen arşiv hata vermeli")
	}
}

func TestArchiveExt(t *testing.T) {
	cases := map[string]string{
		"a.tar.gz": ".tar.gz",
		"a.tgz":    ".tar.gz",
		"a.cbz":    ".cbz",
		"a.zip":    ".zip",
	}
	for in, want := range cases {
		if got := ArchiveExt(in); got != want {
			t.Errorf("ArchiveExt(%s) = %s, istediğim %s", in, got, want)
		}
	}
}

func TestIsArchive(t *testing.T) {
	for _, name := range []string{"a.cbz", "a.zip", "a.tar.gz", "a.tgz", "a.tar"} {
		if !IsArchive(name) {
			t.Errorf("%s arşiv sayılmalı", name)
		}
	}
	if IsArchive("a.png") {
		t.Error("a.png arşiv sayılmamalı")
	}
}
