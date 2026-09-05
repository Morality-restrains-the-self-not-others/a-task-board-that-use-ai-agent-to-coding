package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// Browser PUT to a query-string presigned URL sends these as CORS preflight
// Access-Control-Request-Headers. Do not include Authorization: Bearer is not
// part of the COS signature and COS CORS 403s OPTIONS when it is requested.
func VendorDocsAllowedCORSHeaders() []string {
	return []string{
		"Content-Type",
		"Content-Length",
		"x-cos-server-side-encryption",
		"Origin",
	}
}

func NormalizeCORSOrigins(origins []string) []string {
	seen := make(map[string]struct{}, len(origins))
	out := make([]string, 0, len(origins))
	for _, raw := range origins {
		o := strings.TrimSpace(raw)
		if o == "" {
			continue
		}
		if _, ok := seen[o]; ok {
			continue
		}
		seen[o] = struct{}{}
		out = append(out, o)
	}
	return out
}

func VendorDocsCORSOptions(origins []string) *cos.BucketPutCORSOptions {
	origins = NormalizeCORSOrigins(origins)
	if len(origins) == 0 {
		return nil
	}
	return &cos.BucketPutCORSOptions{
		Rules: []cos.BucketCORSRule{
			{
				ID:             "vendor-docs-browser-put",
				AllowedOrigins: origins,
				AllowedMethods: []string{"PUT", "GET", "HEAD", "POST"},
				AllowedHeaders: VendorDocsAllowedCORSHeaders(),
				ExposeHeaders:  []string{"ETag", "Content-Length", "x-cos-request-id"},
				MaxAgeSeconds:  600,
			},
		},
	}
}

func CORSAllowsAuthorization(opt *cos.BucketPutCORSOptions) bool {
	if opt == nil {
		return false
	}
	for _, rule := range opt.Rules {
		for _, h := range rule.AllowedHeaders {
			if strings.EqualFold(strings.TrimSpace(h), "authorization") {
				return true
			}
		}
	}
	return false
}

func (c *RealCOSAPI) EnsureCORS(ctx context.Context, origins []string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("nil cos client")
	}
	opt := VendorDocsCORSOptions(origins)
	if opt == nil {
		return nil
	}
	_, err := c.client.Bucket.PutCORS(ctx, opt)
	return err
}

func ensureVendorDocsCORS(api *RealCOSAPI, origins []string) {
	opt := VendorDocsCORSOptions(origins)
	if opt == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := api.EnsureCORS(ctx, origins); err != nil {
		slog.Warn("vendor_docs_cors_ensure_failed", "event", "VendorDocsCORSEnsureFailed", "err", err)
		return
	}
	slog.Info("vendor_docs_cors_ensured", "event", "VendorDocsCORSEnsured", "origin_count", len(opt.Rules[0].AllowedOrigins))
}
