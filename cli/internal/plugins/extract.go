package plugins

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractZip распаковывает zip архив в destDir. destDir создаётся (вместе с
// родителями) если не существует. Присутствует защита от zip slip, проверяя что
// итоговый путь действительно находится внутри destDir.
func extractZip(zipPath, destDir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolve dest dir: %w", err)
	}

	for _, f := range zr.File {
		if err := extractZipEntry(f, destAbs); err != nil {
			return err
		}
	}
	return nil
}

// extractZipEntry обрабатывает один файл из zip-архива
func extractZipEntry(f *zip.File, destAbs string) error {
	cleaned := filepath.Clean(f.Name)
	// путь не должен выходить за пределы destAbs.
	if strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) || cleaned == ".." || strings.Contains(cleaned, "\x00") {
		return fmt.Errorf("illegal zip entry path: %q", f.Name)
	}

	target := filepath.Join(destAbs, cleaned)
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve zip entry path: %w", err)
	}
	if !strings.HasPrefix(targetAbs, destAbs+string(os.PathSeparator)) && targetAbs != destAbs {
		return fmt.Errorf("illegal zip entry path: %q escapes destination", f.Name)
	}

	mode := f.Mode()

	if f.FileInfo().IsDir() {
		return os.MkdirAll(targetAbs, dirModeOrDefault(mode))
	}

	// Гарантируем что родительская папка существует
	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("open zip entry: %w", err)
	}
	defer rc.Close()

	// повторная установка перезаписывает старые файлы
	out, err := os.OpenFile(targetAbs, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileModeOrDefault(mode))
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	if _, err := io.Copy(out, rc); err != nil {
		_ = out.Close()
		return fmt.Errorf("write file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	return nil
}

func fileModeOrDefault(m os.FileMode) os.FileMode {
	perm := m.Perm()
	if perm == 0 {
		return 0o644
	}
	if perm&0o111 != 0 {
		// Если был хоть один бит x, гарантируем что владелец может запускать
		return perm | 0o100
	}
	return perm
}

// dirModeOrDefault возвращает права для папки.
// Нужны минимум rwx для владельца чтобы потом туда писать.
func dirModeOrDefault(m os.FileMode) os.FileMode {
	perm := m.Perm()
	if perm == 0 {
		return 0o755
	}
	return perm | 0o700
}
