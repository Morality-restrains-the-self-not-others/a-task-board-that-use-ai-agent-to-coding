package infrastructure

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// SignCOSPostPolicy 按腾讯云 POST Object 文档生成 q-signature。
// https://cloud.tencent.com/document/product/436/14690
func SignCOSPostPolicy(secretKey, keyTime, policyText string) string {
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(keyTime))
	signKey := hex.EncodeToString(mac.Sum(nil))
	sum := sha1.Sum([]byte(policyText))
	stringToSign := hex.EncodeToString(sum[:])
	mac2 := hmac.New(sha1.New, []byte(signKey))
	mac2.Write([]byte(stringToSign))
	return hex.EncodeToString(mac2.Sum(nil))
}

func BuildCOSPostFormFields(secretID, secretKey, bucket, objectKey string, size int64, start, end time.Time) (map[string]string, error) {
	if secretID == "" || secretKey == "" || bucket == "" || objectKey == "" {
		return nil, fmt.Errorf("cos post policy missing credentials or key")
	}
	if size <= 0 {
		return nil, fmt.Errorf("file size out of range")
	}
	keyTime := fmt.Sprintf("%d;%d", start.Unix(), end.Unix())
	policyObj := map[string]any{
		"expiration": end.UTC().Format("2006-01-02T15:04:05.000Z"),
		"conditions": []any{
			map[string]string{"bucket": bucket},
			map[string]string{"key": objectKey},
			map[string]string{"q-sign-algorithm": "sha1"},
			map[string]string{"q-ak": secretID},
			map[string]string{"q-sign-time": keyTime},
			[]any{"eq", "$x-cos-server-side-encryption", "AES256"},
			[]any{"content-length-range", 1, size},
		},
	}
	policyText, err := json.Marshal(policyObj)
	if err != nil {
		return nil, err
	}
	policyB64 := base64.StdEncoding.EncodeToString(policyText)
	sig := SignCOSPostPolicy(secretKey, keyTime, string(policyText))
	return map[string]string{
		"key":                          objectKey,
		"policy":                       policyB64,
		"q-sign-algorithm":             "sha1",
		"q-ak":                         secretID,
		"q-key-time":                   keyTime,
		"q-signature":                  sig,
		"x-cos-server-side-encryption": "AES256",
	}, nil
}

func (c *RealCOSAPI) PresignPost(_ context.Context, key, contentType string, size int64, ttl time.Duration) (string, map[string]string, error) {
	_ = contentType
	if ttl <= 0 {
		ttl = 300 * time.Second
	}
	start := time.Now()
	end := start.Add(ttl)
	fields, err := BuildCOSPostFormFields(c.secretID, c.secretKey, c.bucket, key, size, start, end)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/", c.bucket, c.region), fields, nil
}

func (f *FakeCOS) PresignPost(_ context.Context, key, contentType string, size int64, _ time.Duration) (string, map[string]string, error) {
	_ = contentType
	if size <= 0 {
		return "", nil, fmt.Errorf("file size out of range")
	}
	return "https://fake-cos.example/", map[string]string{
		"key":                          key,
		"x-cos-server-side-encryption": "AES256",
	}, nil
}
