package plugins

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
)

// fakeDownloader пишет заранее заданные байты в writer.
// При needErr true возвращает ошибку после записи части данных.
type fakeDownloader struct {
	payload []byte
	needErr error
}

func (f *fakeDownloader) DownloadArtifact(_ context.Context, _ int64, w io.Writer) (int64, error) {
	if f.needErr != nil {
		// Имитируем сетевую ошибку посреди стрима
		half := len(f.payload) / 2
		n, _ := w.Write(f.payload[:half])
		return int64(n), f.needErr
	}
	n, err := w.Write(f.payload)
	return int64(n), err
}

func TestDownloadVerifiedToHappyPath(t *testing.T) {
	t.Parallel()
	payload := []byte("hello plugin world")
	sum := sha256.Sum256(payload)
	expectedHex := hex.EncodeToString(sum[:])

	var buf bytes.Buffer
	checksum, size, err := downloadVerifiedTo(context.Background(), &fakeDownloader{payload: payload}, 1, &buf)
	if err != nil {
		t.Fatalf("downloadVerifiedTo: %v", err)
	}
	if checksum != expectedHex {
		t.Errorf("checksum = %q, want %q", checksum, expectedHex)
	}
	if size != int64(len(payload)) {
		t.Errorf("size = %d, want %d", size, len(payload))
	}
	if !bytes.Equal(buf.Bytes(), payload) {
		t.Errorf("payload mismatch")
	}
}

func TestDownloadVerifiedToPropagatesErr(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	_, _, err := downloadVerifiedTo(context.Background(), &fakeDownloader{payload: []byte("xxxx"), needErr: errors.New("boom")}, 1, &buf)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error should wrap original: %v", err)
	}
}

func TestVerifyDownloaded(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		actualSize        int64
		actualChecksum    string
		expectedSize      int64
		expectedChecksum  string
		wantErrContaining string
	}{
		{name: "match", actualSize: 10, actualChecksum: "AABB", expectedSize: 10, expectedChecksum: "aabb"},
		{name: "size mismatch", actualSize: 9, actualChecksum: "aabb", expectedSize: 10, expectedChecksum: "aabb", wantErrContaining: "size mismatch"},
		{name: "checksum mismatch", actualSize: 10, actualChecksum: "aacc", expectedSize: 10, expectedChecksum: "aabb", wantErrContaining: "checksum mismatch"},
		{name: "expected size zero is skipped", actualSize: 10, actualChecksum: "aabb", expectedSize: 0, expectedChecksum: "aabb"},
		{name: "expected checksum empty is skipped", actualSize: 10, actualChecksum: "aabb", expectedSize: 10, expectedChecksum: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyDownloaded(tc.actualSize, tc.actualChecksum, tc.expectedSize, tc.expectedChecksum)
			if tc.wantErrContaining == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErrContaining)
			}
			if !strings.Contains(err.Error(), tc.wantErrContaining) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantErrContaining)
			}
		})
	}
}
