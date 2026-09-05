package infrastructure

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"confload"
	dbload "dbload"
	_ "github.com/go-sql-driver/mysql"
)

const (
	DefaultSecretKey = "dev-saas-ai-provider-change-me-in-production"
	JWTIssuer        = "saas-ai-provider"
	DefaultJWTTTL    = 8 * 3600
)

type Config struct {
	SecretKey              string
	Host                   string
	Port                   int
	DatabasePath           string
	SSOJwtSecret           string
	SSOJwtIssuer           string
	SSOAudience            string
	OIDCRpIssuer           string
	OIDCClientID           string
	OIDCClientSecret       string
	OIDCPublicOrigin       string
	OIDCScopes             []string
	CloudServiceBaseURL    string
	InternalSecret         string
	FrontendDistDir        string
	JWTTTLSeconds          int
	IsDev                  bool
	RepoRoot               string
	VendorDocsDir          string // 厂商申请证照本地存储根目录
	VendorDocsBackend      string
	VendorDocsCOS          VendorDocsCOSConfig
	TaskAuthBaseURL        string // 同节点调用 taskAuth；拆分时改 ${subdomains.auth}
	TaskAuthInternalSecret string
}

func FindMonorepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 16; i++ {
		if _, err := os.Stat(filepath.Join(dir, "db", "registry.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("monorepo root not found")
}

func LoadConfig(root string) (*Config, error) {
	cfg := &Config{
		SecretKey:           DefaultSecretKey,
		Host:                "0.0.0.0",
		Port:                8010,
		SSOJwtIssuer:        "task2app-sso",
		SSOAudience:         "saas-ai-provider",
		OIDCClientID:        "ai-provider",
		OIDCClientSecret:    "aip-oidc-dev-secret",
		OIDCScopes:          []string{"openid", "email", "profile"},
		CloudServiceBaseURL: "http://127.0.0.1:8018",
		// 同节点 loopback；拆分时改 ${subdomains.auth}
		TaskAuthBaseURL: "http://127.0.0.1:8003",
		JWTTTLSeconds:   DefaultJWTTTL,
		RepoRoot:        root,
		IsDev:           true,
		VendorDocsDir:   filepath.Join(root, "taskAiProvider", "data", "vendor-docs"),
	}
	// Resolve MySQL DSN from registry
	mysqlDSN, err := dbload.ResolveMySQLDSN("ai-provider", root)
	if err != nil {
		return nil, fmt.Errorf("ai-provider MySQL DSN: %w", err)
	}
	cfg.DatabasePath = mysqlDSN
	cfg.FrontendDistDir = filepath.Join(root, "taskAiProvider", "frontend", "dist")

	if v := strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL")); v != "" {
		cfg.CloudServiceBaseURL = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("DJANGO_SECRET_KEY")); v != "" {
		cfg.SecretKey = v
	}
	if v := strings.TrimSpace(os.Getenv("TASK2APP_SSO_JWT_SECRET")); v != "" {
		cfg.SSOJwtSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("OIDC_RP_ISSUER")); v != "" {
		cfg.OIDCRpIssuer = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("OIDC_RP_CLIENT_SECRET")); v != "" {
		cfg.OIDCClientSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("X_INTERNAL_SECRET")); v != "" {
		cfg.InternalSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_BASE_URL")); v != "" {
		cfg.TaskAuthBaseURL = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_INTERNAL_SECRET")); v != "" {
		cfg.TaskAuthInternalSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("AI_PROVIDER_VENDOR_DOCS_DIR")); v != "" {
		cfg.VendorDocsDir = v
	}

	var raw map[string]any
	if err := confload.ReadAppConfig(root, "ai/ai-provider", &raw); err == nil && raw != nil {
		if h, ok := raw["host"].(string); ok && strings.TrimSpace(h) != "" {
			cfg.Host = strings.TrimSpace(h)
		}
		if p, ok := asInt(raw["port"]); ok {
			cfg.Port = p
		}
		if d, ok := raw["isDev"].(bool); ok {
			cfg.IsDev = d
		}
		if iss, ok := raw["oidcRpIssuer"].(string); ok && strings.TrimSpace(iss) != "" {
			cfg.OIDCRpIssuer = resolvePlaceholders(root, strings.TrimSpace(iss))
		}
		if ah, ok := raw["allowedHost"].(string); ok && strings.TrimSpace(ah) != "" {
			cfg.OIDCPublicOrigin = strings.TrimRight(resolvePlaceholders(root, strings.TrimSpace(ah)), "/")
		}
		if v, ok := raw["taskAuthBaseUrl"].(string); ok && strings.TrimSpace(v) != "" {
			cfg.TaskAuthBaseURL = strings.TrimRight(resolvePlaceholders(root, strings.TrimSpace(v)), "/")
		}
		if v, ok := raw["taskAuthInternalSecret"].(string); ok && strings.TrimSpace(v) != "" {
			cfg.TaskAuthInternalSecret = strings.TrimSpace(v)
		}
		if v, ok := raw["vendorDocsDir"].(string); ok && strings.TrimSpace(v) != "" {
			cfg.VendorDocsDir = resolvePlaceholders(root, strings.TrimSpace(v))
		}
	}
	// OPT-20260810-017：taskAuth 内部密钥与 taskBill 同源，经 sync.manifest 从
	// conf/auth/task-auth/config.yaml 同步到本目录 task-auth.yaml（规则 29 服务仅读本目录）。
	// 优先级：TASKAUTH_INTERNAL_SECRET env > 本文件 taskAuthInternalSecret > task-auth.yaml 片段。
	if cfg.TaskAuthInternalSecret == "" {
		var frag struct {
			InternalSecret string `yaml:"internalSecret"`
		}
		if err := confload.ReadAppFragment(root, "ai/ai-provider", "task-auth.yaml", &frag); err == nil {
			if v := strings.TrimSpace(frag.InternalSecret); v != "" {
				cfg.TaskAuthInternalSecret = v
			}
		}
	}
	// 内部密钥兜底：与网关/其他服务共用 X_INTERNAL_SECRET 或 conf/gateway 注入的 secret
	if cfg.TaskAuthInternalSecret == "" {
		cfg.TaskAuthInternalSecret = cfg.InternalSecret
	}

	if cfg.SSOJwtSecret == "" {
		var sso struct {
			SSOJwtSecret string `yaml:"ssoJwtSecret"`
		}
		if err := confload.ReadAppConfig(root, "core/sso", &sso); err == nil {
			cfg.SSOJwtSecret = strings.TrimSpace(sso.SSOJwtSecret)
		}
		if cfg.SSOJwtSecret == "" {
			var frag struct {
				SSOJwtSecret         string `yaml:"ssoJwtSecret"`
				Task2appSsoJwtSecret string `yaml:"task2appSsoJwtSecret"`
			}
			if err := confload.ReadAppFragment(root, "ai/ai-provider", "django.yaml", &frag); err == nil {
				if v := strings.TrimSpace(frag.SSOJwtSecret); v != "" {
					cfg.SSOJwtSecret = v
				} else if v := strings.TrimSpace(frag.Task2appSsoJwtSecret); v != "" {
					cfg.SSOJwtSecret = v
				}
			}
		}
	}
	if cfg.SSOJwtSecret == "" {
		cfg.SSOJwtSecret = cfg.SecretKey
	}
	if cfg.OIDCRpIssuer == "" {
		cfg.OIDCRpIssuer = resolvePlaceholders(root, "${subdomains.gateway}")
	}
	cfg.OIDCRpIssuer = strings.TrimRight(cfg.OIDCRpIssuer, "/")
	applyVendorDocsConfig(cfg, root)
	return cfg, nil
}

func resolvePlaceholders(root, s string) string {
	return confload.ResolveTemplate(s, confload.ResolveBaseYaml(root))
}

func asInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	default:
		return 0, false
	}
}

type DB struct {
	SQL *sql.DB
}

func OpenDB(dsn string) (*DB, error) {
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return &DB{SQL: sqlDB}, nil
}

func (d *DB) Close() error { return d.SQL.Close() }

func (d *DB) PingOK() (latencyMs int, err error) {
	start := time.Now()
	err = d.SQL.Ping()
	if err != nil {
		return 0, err
	}
	var one int
	err = d.SQL.QueryRow("SELECT 1").Scan(&one)
	return int(time.Since(start).Milliseconds()), err
}

// --- JWT ---

func IssueToken(secret, subject, typ string, ttlSec int) (string, error) {
	now := time.Now().Unix()
	claims := map[string]any{
		"iss": JWTIssuer,
		"sub": subject,
		"typ": typ,
		"iat": now,
		"exp": now + int64(ttlSec),
	}
	return SignHS256(secret, claims)
}

func SignHS256(secret string, claims map[string]any) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	mid := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(header + "." + mid))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return header + "." + mid + "." + sig, nil
}

func ParseHS256JWT(token, secret, wantIss, wantAud, wantTyp string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt format")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("bad jwt sig")
	}
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, fmt.Errorf("bad jwt signature")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("bad jwt payload")
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("bad jwt claims")
	}
	now := time.Now().Unix()
	if exp, ok := claimInt64(claims["exp"]); !ok || now >= exp {
		return nil, fmt.Errorf("jwt expired")
	}
	if _, ok := claimInt64(claims["iat"]); !ok {
		return nil, fmt.Errorf("iat required")
	}
	if strings.TrimSpace(fmt.Sprint(claims["sub"])) == "" || fmt.Sprint(claims["sub"]) == "<nil>" {
		return nil, fmt.Errorf("sub required")
	}
	if wantTyp != "" {
		if typ, _ := claims["typ"].(string); typ != wantTyp {
			return nil, fmt.Errorf("bad typ")
		}
	}
	if wantIss != "" {
		if iss, _ := claims["iss"].(string); iss != wantIss {
			return nil, fmt.Errorf("bad iss")
		}
	}
	if wantAud != "" && !audienceOK(claims["aud"], wantAud) {
		return nil, fmt.Errorf("bad aud")
	}
	return claims, nil
}

func claimInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int64:
		return t, true
	case int:
		return int64(t), true
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case string:
		// 主站 SSO bridge 按 ID 字符串传输规范签发 sub=str(user_id)；须在入库前解析。
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(s, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func audienceOK(aud any, want string) bool {
	switch t := aud.(type) {
	case string:
		return t == want
	case []any:
		for _, v := range t {
			if s, ok := v.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}

func ClaimString(claims map[string]any, key string) string {
	v, ok := claims[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

// --- snowflake-ish ID ---

var idSeq uint32

func NextID() int64 {
	now := time.Now().UnixMilli()
	seq := atomic.AddUint32(&idSeq, 1) & 0xFFF
	return (now << 12) | int64(seq)
}

func RandomSecret(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func RandomPasswordHash() string {
	// Django-compatible enough placeholder; local login disabled
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "pbkdf2_sha256$600000$go$" + base64.StdEncoding.EncodeToString(b)
}

func MustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func EncodeBinaryID(id int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(id))
	return b
}
