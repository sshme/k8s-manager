package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

var ErrSessionNotFound = errors.New("session not found")

type Store struct {
	service string
	entry   string
	path    string
}

func NewStore(service, entry, path string) *Store {
	return &Store{
		service: service,
		entry:   entry,
		path:    path,
	}
}

func (s *Store) Save(session *Session) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key, err := s.loadOrCreateKey()
	if err != nil {
		return err
	}

	ciphertext, err := encrypt(key, payload)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	return os.WriteFile(s.path, []byte(ciphertext), 0o600)
}

func (s *Store) Load() (*Session, error) {
	key, err := keyring.Get(s.service, s.entry)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	encodedCiphertext, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	payload, err := decrypt(key, string(encodedCiphertext))
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *Store) Delete() error {
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	err := keyring.Delete(s.service, s.entry)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}

	return err
}

func (s *Store) loadOrCreateKey() (string, error) {
	key, err := keyring.Get(s.service, s.entry)
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		return "", err
	}

	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		return "", err
	}

	key = base64.RawStdEncoding.EncodeToString(rawKey)
	if err := keyring.Set(s.service, s.entry, key); err != nil {
		return "", err
	}

	return key, nil
}

func encrypt(encodedKey string, plaintext []byte) (string, error) {
	key, err := base64.RawStdEncoding.DecodeString(encodedKey)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(encodedKey, encodedCiphertext string) ([]byte, error) {
	key, err := base64.RawStdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, err
	}

	ciphertext, err := base64.RawStdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("invalid encrypted session payload")
	}

	nonce := ciphertext[:gcm.NonceSize()]
	data := ciphertext[gcm.NonceSize():]

	return gcm.Open(nil, nonce, data, nil)
}
