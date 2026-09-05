package domain

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	VendorDocKindIDCard           = "id_card"
	VendorDocKindBusinessLicense  = "business_license"
	VendorDocKindImageGroupIcon   = "image_group_icon"
	MaxVendorDocBytes             = 5 << 20
	MaxImageGroupIconBytes        = 512 << 10
	VendorDocBackendLocal         = "local"
	VendorDocBackendCOS           = "cos"
	EventVendorDocumentUploaded   = "VendorDocumentUploaded"
	EventVendorDocPathRuleUpdated = "VendorDocPathRuleUpdated"
	EventImageGroupIconUploaded   = "ImageGroupIconUploaded"
	EventImageGroupCreated        = "ImageGroupCreated"
	EventImageGroupUpdated        = "ImageGroupUpdated"
)

var allowedVendorDocExt = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".pdf":  "application/pdf",
}

var allowedImageGroupIconExt = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
}

var pathRulePlaceholders = map[string]struct{}{
	"keyPrefix": {},
	"userId":    {},
	"kind":      {},
	"id":        {},
	"ext":       {},
	"yyyy":      {},
	"mm":        {},
	"dd":        {},
}

func IsVendorKYCKind(kind string) bool {
	return kind == VendorDocKindIDCard || kind == VendorDocKindBusinessLicense
}

func IsVendorDocKind(kind string) bool {
	return IsVendorKYCKind(kind) || kind == VendorDocKindImageGroupIcon
}

func VendorDocContentType(nameOrPath string) string {
	ext := strings.ToLower(extOf(nameOrPath))
	if ct, ok := allowedVendorDocExt[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

func ValidateVendorDocMeta(kind, filename, contentType string, size int64) (ext string, ct string, err error) {
	if kind == VendorDocKindImageGroupIcon {
		return ValidateImageGroupIconMeta(filename, contentType, size)
	}
	if !IsVendorDocKind(kind) {
		return "", "", fmt.Errorf("invalid kind")
	}
	if size <= 0 || size > MaxVendorDocBytes {
		return "", "", fmt.Errorf("file size out of range")
	}
	ext = strings.ToLower(extOf(filename))
	ct, ok := allowedVendorDocExt[ext]
	if !ok {
		return "", "", fmt.Errorf("unsupported file type")
	}
	if contentType != "" && !strings.EqualFold(contentType, ct) {
		return "", "", fmt.Errorf("content type mismatch")
	}
	return ext, ct, nil
}

func ValidateImageGroupIconMeta(filename, contentType string, size int64) (ext string, ct string, err error) {
	if size <= 0 || size > MaxImageGroupIconBytes {
		return "", "", fmt.Errorf("file size out of range")
	}
	ext = strings.ToLower(extOf(filename))
	ct, ok := allowedImageGroupIconExt[ext]
	if !ok {
		return "", "", fmt.Errorf("unsupported file type")
	}
	if contentType != "" && !strings.EqualFold(contentType, ct) {
		return "", "", fmt.Errorf("content type mismatch")
	}
	return ext, ct, nil
}

func ValidatePathRule(pattern string) error {
	p := strings.TrimSpace(pattern)
	if p == "" || strings.Contains(p, "..") || strings.HasPrefix(p, "/") {
		return fmt.Errorf("invalid path rule")
	}
	for _, token := range extractPlaceholders(p) {
		if _, ok := pathRulePlaceholders[token]; !ok {
			return fmt.Errorf("unknown placeholder {%s}", token)
		}
	}
	return nil
}

func RenderPathRule(pattern, keyPrefix string, userID int64, kind, ext string, id int64) (string, error) {
	if err := ValidatePathRule(pattern); err != nil {
		return "", err
	}
	if !IsVendorDocKind(kind) {
		return "", fmt.Errorf("invalid kind")
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	now := time.Now().UTC()
	repl := map[string]string{
		"keyPrefix": strings.Trim(strings.TrimSpace(keyPrefix), "/"),
		"userId":    fmt.Sprintf("%d", userID),
		"kind":      kind,
		"id":        fmt.Sprintf("%d", id),
		"ext":       ext,
		"yyyy":      now.Format("2006"),
		"mm":        now.Format("01"),
		"dd":        now.Format("02"),
	}
	out := pattern
	for k, v := range repl {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	out = strings.Trim(strings.ReplaceAll(out, "\\", "/"), "/")
	if strings.Contains(out, "..") || strings.Contains(out, "{") {
		return "", fmt.Errorf("invalid rendered key")
	}
	return out, nil
}

func OwnsFileKey(userID int64, fileKey string) bool {
	key := strings.TrimSpace(strings.ReplaceAll(fileKey, "\\", "/"))
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return false
	}
	needle := fmt.Sprintf("/%d/", userID)
	return strings.Contains("/"+key+"/", needle)
}

type VendorDocument struct {
	UserID      int64
	Kind        string
	FileKey     string
	ContentType string
	Size        int64
}

type PresignPut struct {
	UploadURL  string
	Method     string
	Headers    map[string]string
	FormFields map[string]string
	ExpiresIn  int
	FileKey    string
}

type VendorDocStore interface {
	Backend() string
	SaveLocal(userID int64, kind, origName string, r io.Reader, size int64) (fileKey string, contentType string, err error)
	PresignPut(ctx context.Context, userID int64, kind, filename, contentType string, size int64) (*PresignPut, error)
	Put(ctx context.Context, userID int64, fileKey, contentType string, r io.Reader, size int64) error
	Head(ctx context.Context, userID int64, fileKey string) error
	Open(ctx context.Context, userID int64, fileKey string) (io.ReadCloser, string, error)
}

type EventBus interface {
	Publish(ctx context.Context, name string, payload map[string]any) error
}

func extOf(name string) string {
	i := strings.LastIndex(name, ".")
	if i < 0 {
		return ""
	}
	return name[i:]
}

func extractPlaceholders(s string) []string {
	var out []string
	for {
		i := strings.Index(s, "{")
		if i < 0 {
			break
		}
		j := strings.Index(s[i:], "}")
		if j < 0 {
			break
		}
		out = append(out, s[i+1:i+j])
		s = s[i+j+1:]
	}
	return out
}
