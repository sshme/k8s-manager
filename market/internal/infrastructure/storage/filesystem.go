package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileStorage implements artifact storage using filesystem
type FileStorage struct {
	basePath string
}

// NewFileStorage creates a new filesystem storage
func NewFileStorage(basePath string) (*FileStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &FileStorage{basePath: basePath}, nil
}

// Save saves an artifact file and returns its storage path and checksum
func (s *FileStorage) Save(pluginID, releaseID int64, osName, arch string, reader io.Reader) (string, string, int64, error) {
	// Create directory structure: plugins/{pluginID}/releases/{releaseID}/{os}/{arch}/
	dir := filepath.Join(s.basePath, fmt.Sprintf("plugins/%d/releases/%d/%s/%s", pluginID, releaseID, osName, arch))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", 0, fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Create file path
	filename := fmt.Sprintf("artifact.bin")
	filePath := filepath.Join(dir, filename)
	
	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	// Calculate checksum while writing
	hash := sha256.New()
	multiWriter := io.MultiWriter(file, hash)
	
	size, err := io.Copy(multiWriter, reader)
	if err != nil {
		os.Remove(filePath)
		return "", "", 0, fmt.Errorf("failed to write file: %w", err)
	}
	
	checksum := hex.EncodeToString(hash.Sum(nil))
	
	// Return relative path from basePath
	relPath, err := filepath.Rel(s.basePath, filePath)
	if err != nil {
		relPath = filePath
	}
	
	return relPath, checksum, size, nil
}

// Get retrieves an artifact file
func (s *FileStorage) Get(storagePath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, storagePath)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// Delete deletes an artifact file
func (s *FileStorage) Delete(storagePath string) error {
	fullPath := filepath.Join(s.basePath, storagePath)
	return os.Remove(fullPath)
}

// Exists checks if an artifact file exists
func (s *FileStorage) Exists(storagePath string) bool {
	fullPath := filepath.Join(s.basePath, storagePath)
	_, err := os.Stat(fullPath)
	return err == nil
}

