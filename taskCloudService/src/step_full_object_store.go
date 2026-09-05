package main

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

	"github.com/tencentyun/cos-go-sdk-v5"
	"tracelog"
)

// StepFullObjectStore is the blob port for step_full.json (COS or memory).
type StepFullObjectStore interface {
	Put(ctx context.Context, key string, body []byte) (etag string, err error)
	Get(ctx context.Context, key string) (body []byte, found bool, err error)
}

type memoryStepFullStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newMemoryStepFullStore() *memoryStepFullStore {
	return &memoryStepFullStore{objects: map[string][]byte{}}
}

func (s *memoryStepFullStore) Put(_ context.Context, key string, body []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.objects == nil {
		s.objects = map[string][]byte{}
	}
	s.objects[key] = append([]byte(nil), body...)
	return "mem", nil
}

func (s *memoryStepFullStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.objects[key]
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), b...), true, nil
}

type realStepFullCOS struct {
	client *cos.Client
}

func newRealStepFullCOS(bucket, region, secretID, secretKey string) (*realStepFullCOS, error) {
	bucket = strings.TrimSpace(bucket)
	region = strings.TrimSpace(region)
	secretID = strings.TrimSpace(secretID)
	secretKey = strings.TrimSpace(secretKey)
	if bucket == "" || region == "" || secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("cos credentials not configured")
	}
	u, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region))
	if err != nil {
		return nil, err
	}
	direct := tracelog.DirectClient(30 * time.Second)
	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Timeout: 30 * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
			Transport: direct.Transport,
		},
	})
	return &realStepFullCOS{client: client}, nil
}

func (c *realStepFullCOS) Put(ctx context.Context, key string, body []byte) (string, error) {
	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType:              "application/json",
			XCosServerSideEncryption: "AES256",
		},
	}
	resp, err := c.client.Object.Put(ctx, key, bytes.NewReader(body), opt)
	if err != nil {
		return "", err
	}
	etag := ""
	if resp != nil && resp.Response != nil {
		etag = strings.Trim(resp.Header.Get("ETag"), `"`)
	}
	return etag, nil
}

func (c *realStepFullCOS) Get(ctx context.Context, key string) ([]byte, bool, error) {
	resp, err := c.client.Object.Get(ctx, key, nil)
	if err != nil {
		if cos.IsNotFoundError(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	if resp == nil || resp.Body == nil {
		return nil, false, nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
	if err != nil {
		return nil, false, err
	}
	return raw, true, nil
}

func (c *realStepFullCOS) Delete(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("cos client not configured")
	}
	_, err := c.client.Object.Delete(ctx, key)
	return err
}

var stepFullObjects StepFullObjectStore = newMemoryStepFullStore()

func resetStepFullObjectsForTest() {
	stepFullObjects = newMemoryStepFullStore()
}

func initStepFullObjectStoreFromCfg() {
	if !strings.EqualFold(strings.TrimSpace(stepFullCOSCfg.Backend), "cos") {
		stepFullObjects = newMemoryStepFullStore()
		return
	}
	real, err := newRealStepFullCOS(
		stepFullCOSCfg.Bucket,
		stepFullCOSCfg.Region,
		stepFullCOSCfg.SecretID,
		stepFullCOSCfg.SecretKey,
	)
	if err != nil {
		tracelog.LogForwardStage(context.Background(), "step_full_cos_init_fallback_memory", map[string]any{
			"error": err.Error(),
		})
		stepFullObjects = newMemoryStepFullStore()
		return
	}
	stepFullObjects = real
	tracelog.LogForwardStage(context.Background(), "step_full_cos_init_ok", map[string]any{
		"bucket": stepFullCOSCfg.Bucket, "region": stepFullCOSCfg.Region,
	})
}
