package infrastructure

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"taskAiProvider/domain"
)

func TestCOSStorePresignAndHead(t *testing.T) {
	fake := NewFakeCOS()
	path := NewVendorDocsPathState(&Config{VendorDocsCOS: VendorDocsCOSConfig{
		KeyPrefix: "vendor-docs",
		PathRule:  defaultVendorDocsRule,
	}})
	st := &COSVendorDocStore{API: fake, Path: path, TTL: 300}
	ps, err := st.PresignPut(context.Background(), 42, domain.VendorDocKindIDCard, "id.png", "image/png", 4)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ps.FileKey, "vendor-docs/42/") || !strings.HasSuffix(ps.FileKey, ".png") {
		t.Fatalf("file_key=%s", ps.FileKey)
	}
	if ps.Method != "POST" || !strings.Contains(ps.UploadURL, "fake-cos.example") {
		t.Fatalf("upload_url must be COS POST Object, got method=%s url=%s", ps.Method, ps.UploadURL)
	}
	if strings.Contains(ps.UploadURL, "local-put") {
		t.Fatalf("COS must not proxy local-put, got %s", ps.UploadURL)
	}
	if ps.FormFields["key"] != ps.FileKey || ps.FormFields["x-cos-server-side-encryption"] != "AES256" {
		t.Fatalf("form_fields=%v", ps.FormFields)
	}
	if ps.Headers["x-cos-server-side-encryption"] != "" {
		t.Fatalf("browser headers must not include SSE: %v", ps.Headers)
	}
	if err := st.Head(context.Background(), 42, ps.FileKey); err == nil {
		t.Fatal("head before put should fail")
	}
	if err := st.Put(context.Background(), 42, ps.FileKey, "image/png", bytes.NewReader([]byte("png!")), 4); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := st.Head(context.Background(), 42, ps.FileKey); err != nil {
		t.Fatalf("head after put: %v", err)
	}
	if err := st.Head(context.Background(), 99, ps.FileKey); err == nil {
		t.Fatal("expected ownership reject")
	}
}

func TestFallbackHeadUsesLocal(t *testing.T) {
	root := t.TempDir()
	key, _, err := SaveVendorDocument(root, 7, VendorDocKindIDCard, "id.jpg", bytes.NewReader([]byte("abcde")), 5)
	if err != nil {
		t.Fatal(err)
	}
	fake := NewFakeCOS()
	st := &FallbackVendorDocStore{
		Primary: &COSVendorDocStore{API: fake, Path: NewVendorDocsPathState(nil), TTL: 300},
		Local:   &LocalVendorDocStore{Root: root},
	}
	if err := st.Head(context.Background(), 7, key); err != nil {
		t.Fatalf("local fallback: %v", err)
	}
}

func TestNewVendorDocStoreLocalWhenNoCOSCreds(t *testing.T) {
	cfg := &Config{VendorDocsBackend: domain.VendorDocBackendCOS, VendorDocsDir: t.TempDir()}
	st := NewVendorDocStore(cfg, NewVendorDocsPathState(cfg), nil)
	if st.Backend() != domain.VendorDocBackendLocal {
		t.Fatalf("backend=%s", st.Backend())
	}
}
