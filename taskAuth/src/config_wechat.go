package main

import (
	"log"
	"os"
	"strings"

	"confload"
)

// loadWeChatConfig 从 conf/auth/task-auth/ 读取微信开放平台 OAuth 配置（v64 多应用）。
// 配置模型: wechat.apps.<key>.{appId,appSecret,redirectUri,type,token,encodingAESKey}
//
//	type=qr       → 开放平台网站应用扫码 (qrconnect, snsapi_login)
//	type=web_oauth → 公众号网页授权 (oauth2/authorize, snsapi_userinfo) — 微信内置浏览器
//	type=mp       → 服务号消息回调（无 redirectUri；Token 必填才启用 callback）
//
// 向后兼容: 旧顶层 wechat.{appId,appSecret,redirectUri} 自动映射为 apps.web。
// 如果配置片段不存在或字段为空，微信登录功能保持关闭（静默降级）。
func loadWeChatConfig(repoRoot string) {
	legacy := WeChatAppConfig{Key: "web", Type: "qr"}
	if v := strings.TrimSpace(os.Getenv("WECHAT_APP_ID")); v != "" {
		legacy.AppID = v
	}
	if v := strings.TrimSpace(os.Getenv("WECHAT_APP_SECRET")); v != "" {
		legacy.AppSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("WECHAT_REDIRECT_URI")); v != "" {
		legacy.RedirectURI = v
	}

	var wrap struct {
		WeChat *struct {
			AppID       string `yaml:"appId"`
			AppSecret   string `yaml:"appSecret"`
			RedirectURI string `yaml:"redirectUri"`
			Apps        map[string]struct {
				AppID          string `yaml:"appId"`
				AppSecret      string `yaml:"appSecret"`
				RedirectURI    string `yaml:"redirectUri"`
				Type           string `yaml:"type"`
				Token          string `yaml:"token"`
				EncodingAESKey string `yaml:"encodingAESKey"`
			} `yaml:"apps"`
			MPEgress *struct {
				BaseURL        string `yaml:"baseUrl"`
				InternalSecret string `yaml:"internalSecret"`
			} `yaml:"mpEgress"`
		} `yaml:"wechat"`
	}
	if err := confload.ReadAppConfig(repoRoot, "auth/task-auth", &wrap); err != nil || wrap.WeChat == nil {
		return
	}
	w := wrap.WeChat

	if len(w.Apps) > 0 {
		apps := map[string]*WeChatAppConfig{}
		for key, a := range w.Apps {
			apps[key] = &WeChatAppConfig{
				Key:            key,
				AppID:          strings.TrimSpace(a.AppID),
				AppSecret:      strings.TrimSpace(a.AppSecret),
				RedirectURI:    strings.TrimSpace(a.RedirectURI),
				Type:           strings.TrimSpace(a.Type),
				Token:          strings.TrimSpace(a.Token),
				EncodingAESKey: strings.TrimSpace(a.EncodingAESKey),
			}
			if apps[key].Type == "" {
				apps[key].Type = "qr"
			}
		}
		if legacy.AppID != "" {
			if b, ok := apps["web"]; ok {
				b.AppID = legacy.AppID
				b.AppSecret = legacy.AppSecret
				b.RedirectURI = legacy.RedirectURI
			} else {
				apps["web"] = &WeChatAppConfig{Key: "web", AppID: legacy.AppID, AppSecret: legacy.AppSecret, RedirectURI: legacy.RedirectURI, Type: "qr"}
			}
		}
		wechatApps = apps
	} else {
		if legacy.AppID == "" && w.AppID != "" {
			legacy.AppID = strings.TrimSpace(w.AppID)
		}
		if legacy.AppSecret == "" && w.AppSecret != "" {
			legacy.AppSecret = strings.TrimSpace(w.AppSecret)
		}
		if legacy.RedirectURI == "" && w.RedirectURI != "" {
			legacy.RedirectURI = strings.TrimSpace(w.RedirectURI)
		}
		if legacy.AppID != "" || legacy.AppSecret != "" || legacy.RedirectURI != "" {
			wechatApps = map[string]*WeChatAppConfig{"web": &legacy}
		}
	}

	subs := confload.ResolveBaseYaml(repoRoot)
	for _, a := range wechatApps {
		a.AppID = confload.ResolveTemplate(a.AppID, subs)
		a.AppSecret = confload.ResolveTemplate(a.AppSecret, subs)
		a.RedirectURI = confload.ResolveTemplate(a.RedirectURI, subs)
		a.Token = confload.ResolveTemplate(a.Token, subs)
		a.EncodingAESKey = confload.ResolveTemplate(a.EncodingAESKey, subs)
	}
	applyWeChatMPEnvOverrides()
	baseURL, egressSecret := "", ""
	if w.MPEgress != nil {
		baseURL = strings.TrimSpace(w.MPEgress.BaseURL)
		egressSecret = strings.TrimSpace(w.MPEgress.InternalSecret)
	}
	applyWeChatMPEgressSettings(baseURL, egressSecret)

	if wechatEnabled() {
		for key, a := range wechatApps {
			if a.Type == "mp" {
				log.Printf("[taskAuth] WeChat app %q enabled: appId=%s type=%s token_set=%t secret_set=%t aes_key_set=%t",
					key, a.AppID, a.Type, strings.TrimSpace(a.Token) != "", strings.TrimSpace(a.AppSecret) != "", strings.TrimSpace(a.EncodingAESKey) != "")
				continue
			}
			log.Printf("[taskAuth] WeChat app %q enabled: appId=%s type=%s redirectUri=%s",
				key, a.AppID, a.Type, a.RedirectURI)
		}
	}
}

// applyWeChatMPEnvOverrides copies WECHAT_MP_* into wechat.apps.mp (secrets stay out of committed YAML).
func applyWeChatMPEnvOverrides() {
	appID := strings.TrimSpace(os.Getenv("WECHAT_MP_APP_ID"))
	secret := strings.TrimSpace(os.Getenv("WECHAT_MP_APP_SECRET"))
	token := strings.TrimSpace(os.Getenv("WECHAT_MP_TOKEN"))
	aesKey := strings.TrimSpace(os.Getenv("WECHAT_MP_ENCODING_AES_KEY"))
	if appID == "" && secret == "" && token == "" && aesKey == "" {
		return
	}
	if wechatApps == nil {
		wechatApps = map[string]*WeChatAppConfig{}
	}
	mp := wechatApps["mp"]
	if mp == nil {
		mp = &WeChatAppConfig{Key: "mp", Type: "mp"}
		wechatApps["mp"] = mp
	}
	if appID != "" {
		mp.AppID = appID
	}
	if secret != "" {
		mp.AppSecret = secret
	}
	if token != "" {
		mp.Token = token
	}
	if aesKey != "" {
		mp.EncodingAESKey = aesKey
	}
	if mp.Type == "" {
		mp.Type = "mp"
	}
}

func applyWeChatMPEgressSettings(baseURL, secret string) {
	if v := strings.TrimSpace(os.Getenv("WECHAT_MP_EGRESS_BASE_URL")); v != "" {
		baseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("WECHAT_MP_EGRESS_INTERNAL_SECRET")); v != "" {
		secret = v
	}
	wechatMPEgressBaseURL = strings.TrimSpace(baseURL)
	wechatMPEgressSecret = strings.TrimSpace(secret)
	if wechatMPEgressBaseURL != "" {
		log.Printf("[taskAuth] WeChat mpEgress enabled baseUrl=%s secret_set=%t",
			wechatMPEgressBaseURL, wechatMPEgressSecret != "")
	}
}
