package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"confload"
	"github.com/redis/go-redis/v9"
)

const (
	relayStartupWFKeyPrefix    = "relay:startup:wf:"
	relayStartupScopeKeyPrefix = "relay:startup:scope:"
	defaultRelaySessionTTLSec  = 7200
	minRelaySessionTTLSec      = 300
)

// RelayStartupSession mirrors Django RedisRelayStartupSessionRepository payload.
type RelayStartupSession struct {
	WorkflowID       string               `json:"workflow_id"`
	Scope            RelayTaskScope       `json:"scope"`
	Phase            string               `json:"phase"`
	RequestID        *string              `json:"request_id"`
	TokenInitialized bool                 `json:"token_initialized"`
	LastStatus       *RelayStatusSnapshot `json:"last_status"`
	LastStatusSeq    *int                 `json:"last_status_seq"`
	FailureReason    string               `json:"failure_reason"`
	CreatedAt        time.Time            `json:"-"`
	UpdatedAt        time.Time            `json:"-"`
	CreatedAtRaw     string               `json:"created_at"`
	UpdatedAtRaw     string               `json:"updated_at"`
}

type RelayTaskScope struct {
	TenantID    string `json:"tenant_id"`
	WorkspaceID string `json:"workspace_id"`
	TaskID      string `json:"task_id"`
	ExpiresAt   string `json:"expires_at"`
}

func (s RelayTaskScope) identityKey() string {
	return s.TenantID + ":" + s.WorkspaceID + ":" + s.TaskID
}

type RelayStatusSnapshot struct {
	Running         bool   `json:"running"`
	OnlineServiceUp bool   `json:"online_service_up"`
	Error           string `json:"error"`
}

func (s RelayStatusSnapshot) convergedRunning() bool {
	return s.Running && s.OnlineServiceUp
}

func (s RelayStatusSnapshot) hasError() bool {
	return strings.TrimSpace(s.Error) != ""
}

// relayStringClient is the Redis string surface used by the session repository.
type relayStringClient interface {
	Get(ctx context.Context, key string) (string, bool, error)
	SetEX(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type goRedisStringClient struct {
	rdb *redis.Client
}

func (c *goRedisStringClient) Get(ctx context.Context, key string) (string, bool, error) {
	raw, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return raw, true, nil
}

func (c *goRedisStringClient) SetEX(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *goRedisStringClient) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

type relaySessionStore struct {
	client relayStringClient
	ttl    time.Duration
}

var relaySessions *relaySessionStore

func workflowKey(workflowID string) string {
	return relayStartupWFKeyPrefix + workflowID
}

func scopeKey(scope RelayTaskScope) string {
	return relayStartupScopeKeyPrefix + scope.identityKey()
}

func initRelayRedis(repoRoot string) {
	// Priority: REDIS_* env > taskCloudService/config.yaml (cfg.*) > domain-events fragment > defaults.
	hostCfg, portCfg, dbCfg := loadRedisConnFromConf(repoRoot)
	host := strings.TrimSpace(os.Getenv("REDIS_HOST"))
	if host == "" {
		host = strings.TrimSpace(cfg.RedisHost)
	}
	if host == "" {
		host = hostCfg
	}
	port := portCfg
	if cfg.RedisPort != 0 {
		port = cfg.RedisPort
	}
	if portRaw := strings.TrimSpace(os.Getenv("REDIS_PORT")); portRaw != "" {
		if p, err := strconv.Atoi(portRaw); err == nil {
			port = p
		}
	}
	db := dbCfg
	if cfg.RedisHost != "" || cfg.RedisPort != 0 {
		db = cfg.RedisDB
	}
	if dbRaw := strings.TrimSpace(os.Getenv("REDIS_DB")); dbRaw != "" {
		if d, err := strconv.Atoi(dbRaw); err == nil {
			db = d
		}
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if port == 0 {
		port = 6379
	}

	ttlSec := defaultRelaySessionTTLSec
	if envTTL := strings.TrimSpace(os.Getenv("RELAY_STARTUP_SESSION_TTL_SECONDS")); envTTL != "" {
		if n, err := strconv.Atoi(envTTL); err == nil {
			ttlSec = n
		}
	}
	if ttlSec < minRelaySessionTTLSec {
		ttlSec = minRelaySessionTTLSec
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", host, port),
		DB:   db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[taskCloudService] relay redis unavailable (%s:%d/%d): %v — converge will fail until Redis is up", host, port, db, err)
	} else {
		log.Printf("[taskCloudService] relay redis: %s:%d db=%d ttl=%ds", host, port, db, ttlSec)
	}
	relaySessions = &relaySessionStore{
		client: &goRedisStringClient{rdb: rdb},
		ttl:    time.Duration(ttlSec) * time.Second,
	}
	cfg.RedisHost = host
	cfg.RedisPort = port
	cfg.RedisDB = db
	cfg.RelaySessionTTLSec = ttlSec
}

func loadRedisConnFromConf(repoRoot string) (host string, port int, db int) {
	host, port, db = "127.0.0.1", 6379, 0
	type redisFrag struct {
		Redis struct {
			Host string `yaml:"host"`
			Port int    `yaml:"port"`
			DB   int    `yaml:"db"`
		} `yaml:"redis"`
	}
	var frag redisFrag
	if err := confloadReadRedis(repoRoot, "domain-events", &frag); err != nil {
		_ = confloadReadRedis(repoRoot, "infra/redis", &frag)
	}
	if h := strings.TrimSpace(frag.Redis.Host); h != "" {
		host = h
	}
	if frag.Redis.Port != 0 {
		port = frag.Redis.Port
	}
	db = frag.Redis.DB
	return host, port, db
}

var confloadReadRedis = func(repoRoot, app string, dest any) error {
	return confload.ReadAppConfigResolved(repoRoot, app, dest)
}

func setRelaySessionStoreForTest(store *relaySessionStore) {
	relaySessions = store
}

func (s *relaySessionStore) findByWorkflowID(ctx context.Context, workflowID string) (*RelayStartupSession, error) {
	raw, ok, err := s.client.Get(ctx, workflowKey(workflowID))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return deserializeRelaySession(raw)
}

func (s *relaySessionStore) findLatestByScope(ctx context.Context, scope RelayTaskScope) (*RelayStartupSession, error) {
	sk := scopeKey(scope)
	wfID, ok, err := s.client.Get(ctx, sk)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	wfID = strings.TrimSpace(wfID)
	if wfID == "" {
		return nil, nil
	}
	session, err := s.findByWorkflowID(ctx, wfID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		_ = s.client.Del(ctx, sk)
		return nil, nil
	}
	return session, nil
}

func (s *relaySessionStore) save(ctx context.Context, session *RelayStartupSession) error {
	payload, err := serializeRelaySession(session)
	if err != nil {
		return err
	}
	wfID := session.WorkflowID
	if err := s.client.SetEX(ctx, workflowKey(wfID), payload, s.ttl); err != nil {
		return err
	}
	return s.client.SetEX(ctx, scopeKey(session.Scope), wfID, s.ttl)
}

func serializeRelaySession(session *RelayStartupSession) (string, error) {
	created := session.CreatedAtRaw
	if created == "" {
		created = session.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	updated := session.UpdatedAtRaw
	if updated == "" {
		updated = session.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	payload := map[string]any{
		"workflow_id":       session.WorkflowID,
		"scope":             session.Scope,
		"phase":             session.Phase,
		"request_id":        session.RequestID,
		"token_initialized": session.TokenInitialized,
		"last_status":       nil,
		"last_status_seq":   session.LastStatusSeq,
		"failure_reason":    session.FailureReason,
		"created_at":        created,
		"updated_at":        updated,
	}
	if session.LastStatus != nil {
		payload["last_status"] = map[string]any{
			"running":           session.LastStatus.Running,
			"online_service_up": session.LastStatus.OnlineServiceUp,
			"error":             session.LastStatus.Error,
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func deserializeRelaySession(raw string) (*RelayStartupSession, error) {
	var data struct {
		WorkflowID       string               `json:"workflow_id"`
		Scope            RelayTaskScope       `json:"scope"`
		Phase            string               `json:"phase"`
		RequestID        *string              `json:"request_id"`
		TokenInitialized bool                 `json:"token_initialized"`
		LastStatus       *RelayStatusSnapshot `json:"last_status"`
		LastStatusSeq    *int                 `json:"last_status_seq"`
		FailureReason    string               `json:"failure_reason"`
		CreatedAt        string               `json:"created_at"`
		UpdatedAt        string               `json:"updated_at"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, fmt.Errorf("relay startup session payload: %w", err)
	}
	if strings.TrimSpace(data.WorkflowID) == "" {
		return nil, fmt.Errorf("relay startup session workflow_id missing")
	}
	if strings.TrimSpace(data.Scope.TenantID) == "" || strings.TrimSpace(data.Scope.WorkspaceID) == "" || strings.TrimSpace(data.Scope.TaskID) == "" {
		return nil, fmt.Errorf("relay startup session scope missing")
	}
	createdAt, err := parseRelaySessionTime(data.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("created_at: %w", err)
	}
	updatedAt, err := parseRelaySessionTime(data.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("updated_at: %w", err)
	}
	if data.LastStatus != nil {
		data.LastStatus.Error = strings.TrimSpace(data.LastStatus.Error)
	}
	return &RelayStartupSession{
		WorkflowID:       data.WorkflowID,
		Scope:            data.Scope,
		Phase:            data.Phase,
		RequestID:        data.RequestID,
		TokenInitialized: data.TokenInitialized,
		LastStatus:       data.LastStatus,
		LastStatusSeq:    data.LastStatusSeq,
		FailureReason:    data.FailureReason,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		CreatedAtRaw:     data.CreatedAt,
		UpdatedAtRaw:     data.UpdatedAt,
	}, nil
}

func parseRelaySessionTime(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, fmt.Errorf("missing")
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	layouts := []string{
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid: %s", s)
}
