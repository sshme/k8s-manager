package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"strings"
)

// artifactDownloader - абстракция над market.Service.DownloadArtifact
type artifactDownloader interface {
	DownloadArtifact(ctx context.Context, artifactID int64, w io.Writer) (int64, error)
}

// downloadVerifiedTo стримит артефакт в dst, попутно считая SHA-256 и размер.
// Возвращает вычисленный hex checksum и количество записанных байт.
// Сама верификация checksum и size делается вызывающим кодом.
func downloadVerifiedTo(ctx context.Context, svc artifactDownloader, artifactID int64, dst io.Writer) (string, int64, error) {
	h := sha256.New()
	counter := &countingWriter{}
	mw := io.MultiWriter(dst, h, counter)

	if _, err := svc.DownloadArtifact(ctx, artifactID, mw); err != nil {
		return "", 0, fmt.Errorf("download artifact: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), counter.n, nil
}

// verifyDownloaded проверяет совпадение размера и checksum с тем, что ожидается
func verifyDownloaded(actualSize int64, actualChecksum string, expectedSize int64, expectedChecksum string) error {
	if expectedSize > 0 && actualSize != expectedSize {
		return fmt.Errorf("size mismatch: server reported %d bytes, got %d", expectedSize, actualSize)
	}

	expected := strings.ToLower(strings.TrimSpace(expectedChecksum))
	actual := strings.ToLower(actualChecksum)
	if expected == "" {
		return nil
	}
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

// countingWriter считает суммарное число записанных байт. Используется в io.MultiWriter.
type countingWriter struct {
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}

// hashHex возвращает hex от текущего состояния hash
func hashHex(h hash.Hash) string {
	return hex.EncodeToString(h.Sum(nil))
}
