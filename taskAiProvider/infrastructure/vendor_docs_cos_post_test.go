package infrastructure

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"tracelog"
)

func TestBuildCOSPostFormFields(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	end := start.Add(5 * time.Minute)
	fields, err := BuildCOSPostFormFields("AKIDtest", "sk-test", "ai-provider-1259712831", "vendor-docs/42/id.png", 8, start, end)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"key", "policy", "q-sign-algorithm", "q-ak", "q-key-time", "q-signature", "x-cos-server-side-encryption"} {
		if fields[k] == "" {
			t.Fatalf("missing %s in %v", k, fields)
		}
	}
	if fields["key"] != "vendor-docs/42/id.png" || fields["q-ak"] != "AKIDtest" {
		t.Fatalf("fields=%v", fields)
	}
	if fields["x-cos-server-side-encryption"] != "AES256" {
		t.Fatalf("sse=%s", fields["x-cos-server-side-encryption"])
	}
	raw, err := base64.StdEncoding.DecodeString(fields["policy"])
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	if SignCOSPostPolicy("sk-test", fields["q-key-time"], string(raw)) != fields["q-signature"] {
		t.Fatal("signature mismatch vs policy text")
	}
}

func TestLiveCOSPostObject(t *testing.T) {
	if os.Getenv("LIVE_COS_POST") != "1" {
		t.Skip("set LIVE_COS_POST=1 to POST Object to the configured bucket")
	}
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	api, err := NewRealCOSAPI(cfg)
	if err != nil {
		t.Fatal(err)
	}
	key := "vendor-docs/live-post-probe/icon.png"
	body := []byte("png-live")
	ttl := 5 * time.Minute
	u, fields, err := api.PresignPost(context.Background(), key, "image/png", int64(len(body)), ttl)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, k := range []string{"key", "policy", "q-sign-algorithm", "q-ak", "q-key-time", "q-signature", "x-cos-server-side-encryption"} {
		if err := w.WriteField(k, fields[k]); err != nil {
			t.Fatal(err)
		}
	}
	fw, err := w.CreateFormFile("file", "icon.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, u, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := tracelog.DirectClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Fatalf("POST Object %d %s", resp.StatusCode, msg)
	}
	ct, n, err := api.Head(context.Background(), key)
	if err != nil {
		t.Fatalf("head after post: %v", err)
	}
	if n != int64(len(body)) {
		t.Fatalf("size=%d ct=%s", n, ct)
	}
	if !strings.Contains(u, ".myqcloud.com/") {
		t.Fatalf("upload url=%s", u)
	}
}
