package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"taskAiProvider/domain"

	"github.com/tencentyun/cos-go-sdk-v5"
	"tracelog"
)

// COSAPI 证照对象存储端口（测试注入 FakeCOS）。
type COSAPI interface {
	PresignPost(ctx context.Context, key, contentType string, size int64, ttl time.Duration) (uploadURL string, formFields map[string]string, err error)
	PutObject(ctx context.Context, key, contentType string, r io.Reader, size int64) error
	Head(ctx context.Context, key string) (contentType string, size int64, err error)
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
}

type COSVendorDocStore struct {
	API  COSAPI
	Path *VendorDocsPathState
	TTL  int
}

func (s *COSVendorDocStore) Backend() string { return domain.VendorDocBackendCOS }

func (s *COSVendorDocStore) SaveLocal(int64, string, string, io.Reader, int64) (string, string, error) {
	return "", "", fmt.Errorf("cos backend does not accept multipart")
}

func (s *COSVendorDocStore) PresignPut(ctx context.Context, userID int64, kind, filename, contentType string, size int64) (*domain.PresignPut, error) {
	ext, ct, err := domain.ValidateVendorDocMeta(kind, filename, contentType, size)
	if err != nil {
		return nil, err
	}
	prefix, rule := defaultVendorDocsPref, defaultVendorDocsRule
	if s.Path != nil {
		prefix, rule = s.Path.Get()
	}
	key, err := domain.RenderPathRule(rule, prefix, userID, kind, ext, NextID())
	if err != nil {
		return nil, err
	}
	ttl := time.Duration(s.TTL) * time.Second
	if ttl <= 0 {
		ttl = 300 * time.Second
	}
	u, fields, err := s.API.PresignPost(ctx, key, ct, size, ttl)
	if err != nil {
		return nil, err
	}
	return &domain.PresignPut{
		UploadURL:  u,
		Method:     http.MethodPost,
		Headers:    map[string]string{},
		FormFields: fields,
		ExpiresIn:  int(ttl.Seconds()),
		FileKey:    key,
	}, nil
}

func (s *COSVendorDocStore) Put(ctx context.Context, userID int64, fileKey, contentType string, r io.Reader, size int64) error {
	if !domain.OwnsFileKey(userID, fileKey) {
		return fmt.Errorf("file key ownership mismatch")
	}
	if size <= 0 || size > domain.MaxVendorDocBytes {
		return fmt.Errorf("file size out of range")
	}
	return s.API.PutObject(ctx, fileKey, contentType, r, size)
}

func (s *COSVendorDocStore) Head(ctx context.Context, userID int64, fileKey string) error {
	if !domain.OwnsFileKey(userID, fileKey) {
		return fmt.Errorf("file key ownership mismatch")
	}
	ct, size, err := s.API.Head(ctx, fileKey)
	if err != nil {
		return err
	}
	if size <= 0 || size > domain.MaxVendorDocBytes {
		return fmt.Errorf("file size out of range")
	}
	want := domain.VendorDocContentType(fileKey)
	if ct != "" && want != "application/octet-stream" && mediaType(ct) != mediaType(want) {
		return fmt.Errorf("content type mismatch")
	}
	return nil
}

func (s *COSVendorDocStore) Open(ctx context.Context, userID int64, fileKey string) (io.ReadCloser, string, error) {
	if !domain.OwnsFileKey(userID, fileKey) {
		return nil, "", fmt.Errorf("file key ownership mismatch")
	}
	return s.API.Get(ctx, fileKey)
}

// FallbackVendorDocStore：COS miss 回退本地存量 key。
type FallbackVendorDocStore struct {
	Primary domain.VendorDocStore
	Local   *LocalVendorDocStore
}

func (s *FallbackVendorDocStore) Backend() string { return s.Primary.Backend() }

func (s *FallbackVendorDocStore) SaveLocal(userID int64, kind, origName string, r io.Reader, size int64) (string, string, error) {
	if s.Local != nil && s.Primary.Backend() != domain.VendorDocBackendCOS {
		return s.Local.SaveLocal(userID, kind, origName, r, size)
	}
	return s.Primary.SaveLocal(userID, kind, origName, r, size)
}

func (s *FallbackVendorDocStore) PresignPut(ctx context.Context, userID int64, kind, filename, contentType string, size int64) (*domain.PresignPut, error) {
	return s.Primary.PresignPut(ctx, userID, kind, filename, contentType, size)
}

func (s *FallbackVendorDocStore) Put(ctx context.Context, userID int64, fileKey, contentType string, r io.Reader, size int64) error {
	return s.Primary.Put(ctx, userID, fileKey, contentType, r, size)
}

func (s *FallbackVendorDocStore) Head(ctx context.Context, userID int64, fileKey string) error {
	err := s.Primary.Head(ctx, userID, fileKey)
	if err == nil {
		return nil
	}
	if s.Local != nil {
		return s.Local.Head(ctx, userID, fileKey)
	}
	return err
}

func (s *FallbackVendorDocStore) Open(ctx context.Context, userID int64, fileKey string) (io.ReadCloser, string, error) {
	rc, ct, err := s.Primary.Open(ctx, userID, fileKey)
	if err == nil {
		return rc, ct, nil
	}
	if s.Local != nil {
		return s.Local.Open(ctx, userID, fileKey)
	}
	return nil, "", err
}

func mediaType(ct string) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
}

func NewVendorDocStore(cfg *Config, path *VendorDocsPathState, api COSAPI) domain.VendorDocStore {
	local := &LocalVendorDocStore{Root: "", Path: path}
	if cfg != nil {
		local.Root = cfg.VendorDocsDir
	}
	backend := domain.VendorDocBackendLocal
	if cfg != nil && strings.EqualFold(strings.TrimSpace(cfg.VendorDocsBackend), domain.VendorDocBackendCOS) {
		backend = domain.VendorDocBackendCOS
	}
	if backend != domain.VendorDocBackendCOS {
		return local
	}
	if api == nil && cfg != nil {
		real, err := NewRealCOSAPI(cfg)
		if err == nil {
			api = real
		}
	}
	if api == nil {
		return local
	}
	ttl := 300
	if cfg != nil && cfg.VendorDocsCOS.PresignTTLSeconds > 0 {
		ttl = cfg.VendorDocsCOS.PresignTTLSeconds
	}
	return &FallbackVendorDocStore{
		Primary: &COSVendorDocStore{API: api, Path: path, TTL: ttl},
		Local:   local,
	}
}

type RealCOSAPI struct {
	client    *cos.Client
	secretID  string
	secretKey string
	bucket    string
	region    string
}

func NewRealCOSAPI(cfg *Config) (*RealCOSAPI, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	sid := strings.TrimSpace(cfg.VendorDocsCOS.SecretID)
	sk := strings.TrimSpace(cfg.VendorDocsCOS.SecretKey)
	if sid == "" || sk == "" {
		return nil, fmt.Errorf("cos credentials not configured")
	}
	bucket := strings.TrimSpace(cfg.VendorDocsCOS.Bucket)
	region := strings.TrimSpace(cfg.VendorDocsCOS.Region)
	u, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region))
	if err != nil {
		return nil, err
	}
	direct := tracelog.DirectClient(30 * time.Second)
	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Timeout: 30 * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  sid,
			SecretKey: sk,
			Transport: direct.Transport,
		},
	})
	api := &RealCOSAPI{client: client, secretID: sid, secretKey: sk, bucket: bucket, region: region}
	ensureVendorDocsCORS(api, cfg.VendorDocsCOS.CORSAllowedOrigins)
	return api, nil
}

func (c *RealCOSAPI) PutObject(ctx context.Context, key, contentType string, r io.Reader, size int64) error {
	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType:              contentType,
			ContentLength:            size,
			XCosServerSideEncryption: "AES256",
		},
	}
	_, err := c.client.Object.Put(ctx, key, r, opt)
	return err
}

func (c *RealCOSAPI) Head(ctx context.Context, key string) (string, int64, error) {
	resp, err := c.client.Object.Head(ctx, key, nil)
	if err != nil {
		return "", 0, err
	}
	ct := ""
	var n int64
	if resp != nil && resp.Response != nil {
		ct = resp.Header.Get("Content-Type")
		n = resp.ContentLength
	}
	return ct, n, nil
}

func (c *RealCOSAPI) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	resp, err := c.client.Object.Get(ctx, key, nil)
	if err != nil {
		return nil, "", err
	}
	ct := "application/octet-stream"
	if resp != nil && resp.Response != nil {
		if v := resp.Header.Get("Content-Type"); v != "" {
			ct = v
		}
		return resp.Body, ct, nil
	}
	return io.NopCloser(bytes.NewReader(nil)), ct, nil
}

type FakeCOS struct {
	mu      sync.Mutex
	Objects map[string][]byte
	Types   map[string]string
}

func NewFakeCOS() *FakeCOS {
	return &FakeCOS{Objects: map[string][]byte{}, Types: map[string]string{}}
}

func (f *FakeCOS) PutObject(_ context.Context, key, contentType string, r io.Reader, size int64) error {
	if size <= 0 {
		return fmt.Errorf("file size out of range")
	}
	b, err := io.ReadAll(io.LimitReader(r, size+1))
	if err != nil {
		return err
	}
	if int64(len(b)) != size {
		return fmt.Errorf("size mismatch")
	}
	f.Put(key, contentType, b)
	return nil
}

func (f *FakeCOS) Put(key, contentType string, body []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Objects[key] = append([]byte(nil), body...)
	f.Types[key] = contentType
}

func (f *FakeCOS) Head(_ context.Context, key string) (string, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.Objects[key]
	if !ok {
		return "", 0, fmt.Errorf("not found")
	}
	return f.Types[key], int64(len(b)), nil
}

func (f *FakeCOS) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.Objects[key]
	if !ok {
		return nil, "", fmt.Errorf("not found")
	}
	ct := f.Types[key]
	if ct == "" {
		ct = domain.VendorDocContentType(key)
	}
	return io.NopCloser(bytes.NewReader(b)), ct, nil
}
