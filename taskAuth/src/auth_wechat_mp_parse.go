package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"strings"

	"taskAuth/domain"
)

type wechatMPXML struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   string   `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Event        string   `xml:"Event"`
	EventKey     string   `xml:"EventKey"`
	UnionID      string   `xml:"UnionID"`
	UnionId      string   `xml:"UnionId"`
	Encrypt      string   `xml:"Encrypt"`
}

type wechatMPMessage struct {
	ToUserName   string
	FromUserName string
	CreateTime   string
	MsgType      string
	Event        string
	EventKey     string
	UnionID      string
	Encrypt      string
}

func parseWechatMPXML(body []byte) (wechatMPMessage, error) {
	var raw wechatMPXML
	dec := xml.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&raw); err != nil {
		return wechatMPMessage{}, fmt.Errorf("wechat mp xml: %w", err)
	}
	union := strings.TrimSpace(raw.UnionID)
	if union == "" {
		union = strings.TrimSpace(raw.UnionId)
	}
	return wechatMPMessage{
		ToUserName:   strings.TrimSpace(raw.ToUserName),
		FromUserName: strings.TrimSpace(raw.FromUserName),
		CreateTime:   strings.TrimSpace(raw.CreateTime),
		MsgType:      strings.TrimSpace(raw.MsgType),
		Event:        strings.TrimSpace(raw.Event),
		EventKey:     strings.TrimSpace(raw.EventKey),
		UnionID:      union,
		Encrypt:      strings.TrimSpace(raw.Encrypt),
	}, nil
}

func wechatMPApp() *WeChatAppConfig {
	if wechatApps == nil {
		return nil
	}
	a := wechatApps["mp"]
	if a == nil || strings.TrimSpace(a.AppID) == "" || strings.TrimSpace(a.Token) == "" {
		return nil
	}
	return a
}

func decryptWechatMPAES(encodingAESKey, cipherB64 string) ([]byte, error) {
	key, err := wechatMPAESKey(encodingAESKey)
	if err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return nil, fmt.Errorf("wechat mp encrypt b64: %w", err)
	}
	if len(raw) < aes.BlockSize || len(raw)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("wechat mp encrypt length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, key[:aes.BlockSize]).CryptBlocks(plain, raw)
	plain, err = pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return nil, err
	}
	if len(plain) < 20 {
		return nil, fmt.Errorf("wechat mp decrypt too short")
	}
	msgLen := binary.BigEndian.Uint32(plain[16:20])
	end := 20 + int(msgLen)
	if end > len(plain) {
		return nil, fmt.Errorf("wechat mp decrypt msgLen")
	}
	return plain[20:end], nil
}

func wechatMPAESKey(encodingAESKey string) ([]byte, error) {
	s := strings.TrimSpace(encodingAESKey)
	if s == "" {
		return nil, fmt.Errorf("empty encodingAESKey")
	}
	if !strings.HasSuffix(s, "=") {
		s += "="
	}
	key, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("encodingAESKey b64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("encodingAESKey must decode to 32 bytes")
	}
	return key, nil
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("pkcs7 length")
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, fmt.Errorf("pkcs7 pad")
	}
	for i := 0; i < pad; i++ {
		if data[len(data)-1-i] != byte(pad) {
			return nil, fmt.Errorf("pkcs7 pad mismatch")
		}
	}
	return data[:len(data)-pad], nil
}

func wechatMPMsgSignatureOK(token, timestamp, nonce, encrypt, signature string) bool {
	return domain.WechatMPCheckMsgSignature(token, timestamp, nonce, encrypt, signature)
}
