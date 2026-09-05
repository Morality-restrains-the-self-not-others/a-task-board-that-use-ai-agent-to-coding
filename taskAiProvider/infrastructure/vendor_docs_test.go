package infrastructure

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndResolveVendorDocument(t *testing.T) {
	root := t.TempDir()
	key, ct, err := SaveVendorDocument(root, 42, VendorDocKindIDCard, "id.PNG", bytes.NewReader([]byte("png-bytes")), 9)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if ct != "image/png" {
		t.Fatalf("ct=%s", ct)
	}
	if !strings.HasPrefix(key, "42/") || !strings.HasSuffix(key, ".png") {
		t.Fatalf("key=%s", key)
	}
	abs, err := ResolveVendorDocumentAbs(root, 42, key)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	b, err := os.ReadFile(abs)
	if err != nil || string(b) != "png-bytes" {
		t.Fatalf("read abs=%s err=%v body=%q", abs, err, b)
	}
	if _, err := ResolveVendorDocumentAbs(root, 99, key); err == nil {
		t.Fatal("expected ownership mismatch")
	}
	if _, err := ResolveVendorDocumentAbs(root, 42, "../etc/passwd"); err == nil {
		t.Fatal("expected path escape reject")
	}
}

func TestSaveVendorDocumentRejectsBadTypeAndSize(t *testing.T) {
	root := t.TempDir()
	if _, _, err := SaveVendorDocument(root, 1, VendorDocKindIDCard, "x.exe", bytes.NewReader([]byte("x")), 1); err == nil {
		t.Fatal("expected bad type")
	}
	big := bytes.Repeat([]byte("a"), MaxVendorDocBytes+1)
	if _, _, err := SaveVendorDocument(root, 1, VendorDocKindBusinessLicense, "a.pdf", bytes.NewReader(big), int64(len(big))); err == nil {
		t.Fatal("expected size reject")
	}
	_ = filepath.Join(root, "noop")
}
