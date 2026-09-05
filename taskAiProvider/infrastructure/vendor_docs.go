package infrastructure

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"taskAiProvider/domain"
)

const (
	VendorDocKindIDCard          = domain.VendorDocKindIDCard
	VendorDocKindBusinessLicense = domain.VendorDocKindBusinessLicense
	MaxVendorDocBytes            = domain.MaxVendorDocBytes
)

func IsVendorDocKind(kind string) bool { return domain.IsVendorDocKind(kind) }

func IsVendorKYCKind(kind string) bool { return domain.IsVendorKYCKind(kind) }

func VendorDocContentType(path string) string { return domain.VendorDocContentType(path) }

// SaveVendorDocument 本地落盘（backend=local / 测试）。
func SaveVendorDocument(root string, userID int64, kind, origName string, r io.Reader, size int64) (fileKey string, contentType string, err error) {
	return (&LocalVendorDocStore{Root: root}).SaveLocal(userID, kind, origName, r, size)
}

func ResolveVendorDocumentAbs(root string, userID int64, fileKey string) (string, error) {
	if !domain.OwnsFileKey(userID, fileKey) {
		return "", fmt.Errorf("file key ownership mismatch")
	}
	key := strings.TrimSpace(strings.ReplaceAll(fileKey, "\\", "/"))
	if strings.HasPrefix(key, "/") || strings.Contains(key, "..") {
		return "", fmt.Errorf("invalid file key")
	}
	abs := filepath.Join(root, filepath.FromSlash(key))
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	cleanAbs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(cleanAbs, cleanRoot+string(os.PathSeparator)) && cleanAbs != cleanRoot {
		return "", fmt.Errorf("path escape")
	}
	st, err := os.Stat(cleanAbs)
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("file not found")
	}
	return cleanAbs, nil
}

type LocalVendorDocStore struct {
	Root string
	Path *VendorDocsPathState
}

func (s *LocalVendorDocStore) Backend() string { return domain.VendorDocBackendLocal }

func (s *LocalVendorDocStore) SaveLocal(userID int64, kind, origName string, r io.Reader, size int64) (string, string, error) {
	ext, ct, err := domain.ValidateVendorDocMeta(kind, origName, "", size)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(s.Root) == "" {
		return "", "", fmt.Errorf("vendor docs dir not configured")
	}
	key, err := s.renderKey(userID, kind, ext)
	if err != nil {
		return "", "", err
	}
	abs := filepath.Join(s.Root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return "", "", err
	}
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	written, err := io.Copy(f, io.LimitReader(r, domain.MaxVendorDocBytes+1))
	if err != nil {
		_ = os.Remove(abs)
		return "", "", err
	}
	if written > domain.MaxVendorDocBytes {
		_ = os.Remove(abs)
		return "", "", fmt.Errorf("file size out of range")
	}
	return key, ct, nil
}

func (s *LocalVendorDocStore) PresignPut(_ context.Context, userID int64, kind, filename, contentType string, size int64) (*domain.PresignPut, error) {
	ext, ct, err := domain.ValidateVendorDocMeta(kind, filename, contentType, size)
	if err != nil {
		return nil, err
	}
	key, err := s.renderKey(userID, kind, ext)
	if err != nil {
		return nil, err
	}
	q := url.Values{"file_key": {key}}
	return &domain.PresignPut{
		UploadURL: "/api/ai-provider/vendor-application/local-put/?" + q.Encode(),
		Method:    "PUT",
		Headers:   map[string]string{"Content-Type": ct, "Content-Length": fmt.Sprintf("%d", size)},
		ExpiresIn: 300,
		FileKey:   key,
	}, nil
}

func (s *LocalVendorDocStore) Put(_ context.Context, userID int64, fileKey, contentType string, r io.Reader, size int64) error {
	return s.PutLocal(userID, fileKey, contentType, r, size)
}

func (s *LocalVendorDocStore) Head(_ context.Context, userID int64, fileKey string) error {
	_, err := ResolveVendorDocumentAbs(s.Root, userID, fileKey)
	return err
}

func (s *LocalVendorDocStore) Open(_ context.Context, userID int64, fileKey string) (io.ReadCloser, string, error) {
	abs, err := ResolveVendorDocumentAbs(s.Root, userID, fileKey)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, "", err
	}
	return f, domain.VendorDocContentType(abs), nil
}

func (s *LocalVendorDocStore) PutLocal(userID int64, fileKey, contentType string, r io.Reader, size int64) error {
	if !domain.OwnsFileKey(userID, fileKey) {
		return fmt.Errorf("file key ownership mismatch")
	}
	if size <= 0 || size > domain.MaxVendorDocBytes {
		return fmt.Errorf("file size out of range")
	}
	abs := filepath.Join(s.Root, filepath.FromSlash(fileKey))
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, io.LimitReader(r, domain.MaxVendorDocBytes+1))
	_ = contentType
	return err
}

func (s *LocalVendorDocStore) renderKey(userID int64, kind, ext string) (string, error) {
	prefix, rule := "", "{userId}/{kind}_{id}{ext}"
	if s.Path != nil {
		prefix, rule = s.Path.Get()
	}
	return domain.RenderPathRule(rule, prefix, userID, kind, ext, NextID())
}
