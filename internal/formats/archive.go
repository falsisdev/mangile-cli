package formats

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func IsArchive(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".cbz" || ext == ".zip" || ext == ".tar.gz" || ext == ".tgz" || ext == ".tar"
}

func ArchiveExt(name string) string {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return ".tar.gz"
	}
	return strings.ToLower(filepath.Ext(lower))
}

func ExpandArchive(path, dst string) error {
	data, err := os.Open(path)
	if err != nil {
		return err
	}
	defer data.Close()

	head := make([]byte, 4)
	size, _ := io.ReadFull(data, head)
	head = head[:size]

	switch {
	case size >= 4 && bytes.Equal(head, []byte("PK\x03\x04")):
		return expandZip(path, dst)
	case size >= 3 && bytes.Equal(head[:3], []byte{0x1f, 0x8b, 0x08}):
		ext := ArchiveExt(path)
		if ext == ".tar.gz" || ext == ".tgz" {
			return expandTarGz(data, dst)
		}
		return fmt.Errorf("desteklenmeyen gzip arşivi: %s", path)
	case size >= 2 && bytes.Equal(head[:2], []byte{0x1f, 0x9d}):
		return fmt.Errorf("compress gzip desteklenmiyor: %s", path)
	}
	return fmt.Errorf("tanınmayan arşiv formatı: %s", path)
}

func expandZip(path, dst string) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if strings.Contains(f.Name, "..") {
			continue
		}
		target := filepath.Join(dst, f.Name)
		if err := writeArchiveFile(f.Open, target, f.Mode()); err != nil {
			return err
		}
	}
	return nil
}

func expandTarGz(r io.Reader, dst string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if strings.Contains(hdr.Name, "..") {
			continue
		}
		target := filepath.Join(dst, hdr.Name)
		data, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		mode := os.FileMode(hdr.Mode) & 0o777
		if mode == 0 {
			mode = 0o644
		}
		if err := writeFileSafe(target, data, mode); err != nil {
			return err
		}
	}
}

func writeArchiveFile(open func() (io.ReadCloser, error), target string, mode os.FileMode) error {
	rc, err := open()
	if err != nil {
		return err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	return writeFileSafe(target, data, mode.Perm())
}

func writeFileSafe(target string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, perm)
}
