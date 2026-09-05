package infrastructure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
)

// FernetKeyFromSecret mirrors Python gitOauth token_crypto:
// key = urlsafe_b64(sha256(SECRET_KEY)).
func FernetKeyFromSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	enc := base64.URLEncoding.EncodeToString(sum[:])
	raw, err := base64.URLEncoding.DecodeString(enc)
	if err != nil {
		// URLEncoding always pads; fall back to raw digest.
		return sum[:]
	}
	return raw
}

type Fernet struct {
	signingKey []byte
	encKey     []byte
}

func NewFernet(secret string) *Fernet {
	key := FernetKeyFromSecret(secret)
	if len(key) != 32 {
		sum := sha256.Sum256([]byte(secret))
		key = sum[:]
	}
	return &Fernet{
		signingKey: key[:16],
		encKey:     key[16:],
	}
}

func (f *Fernet) Encrypt(plain string) (string, error) {
	text := strings.TrimSpace(plain)
	if text == "" {
		return "", nil
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	block, err := aes.NewCipher(f.encKey)
	if err != nil {
		return "", err
	}
	padded := pkcs7Pad([]byte(text), aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	token := make([]byte, 0, 1+8+16+len(ciphertext)+32)
	token = append(token, 0x80)
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(time.Now().Unix()))
	token = append(token, ts...)
	token = append(token, iv...)
	token = append(token, ciphertext...)

	mac := hmac.New(sha256.New, f.signingKey)
	mac.Write(token)
	token = append(token, mac.Sum(nil)...)
	return base64.URLEncoding.EncodeToString(token), nil
}

func (f *Fernet) Decrypt(cipherText string) (string, error) {
	text := strings.TrimSpace(cipherText)
	if text == "" {
		return "", nil
	}
	raw, err := base64.URLEncoding.DecodeString(text)
	if err != nil {
		// try RawURLEncoding (no padding)
		raw, err = base64.RawURLEncoding.DecodeString(text)
		if err != nil {
			return "", nil
		}
	}
	if len(raw) < 1+8+16+32 {
		return "", nil
	}
	if raw[0] != 0x80 {
		return "", nil
	}
	body := raw[:len(raw)-32]
	sig := raw[len(raw)-32:]
	mac := hmac.New(sha256.New, f.signingKey)
	mac.Write(body)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", nil
	}
	iv := body[9:25]
	ct := body[25:]
	if len(ct)%aes.BlockSize != 0 {
		return "", nil
	}
	block, err := aes.NewCipher(f.encKey)
	if err != nil {
		return "", err
	}
	plain := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, ct)
	out, err := pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return "", nil
	}
	return string(out), nil
}

func pkcs7Pad(b []byte, blockSize int) []byte {
	pad := blockSize - len(b)%blockSize
	out := make([]byte, len(b)+pad)
	copy(out, b)
	for i := len(b); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}

func pkcs7Unpad(b []byte, blockSize int) ([]byte, error) {
	if len(b) == 0 || len(b)%blockSize != 0 {
		return nil, errors.New("bad padding size")
	}
	pad := int(b[len(b)-1])
	if pad == 0 || pad > blockSize || pad > len(b) {
		return nil, fmt.Errorf("bad pad %d", pad)
	}
	for i := 0; i < pad; i++ {
		if b[len(b)-1-i] != byte(pad) {
			return nil, errors.New("bad padding")
		}
	}
	return b[:len(b)-pad], nil
}
