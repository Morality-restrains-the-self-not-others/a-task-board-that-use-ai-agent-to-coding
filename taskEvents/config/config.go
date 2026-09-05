// Package config loads domainEvents from conf/ YAML.
package config

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"taskEvents/domain"
)

// resolveTemplate expands ${VAR} and ${VAR:-default} patterns using environment
// variables. Shell-style ${VAR:-default} is supported; bare ${VAR} resolves to
// the env value or empty string.
func resolveTemplate(s string) string {
	re := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		varName := sub[1]
		defaultVal := sub[2]
		if val, ok := os.LookupEnv(varName); ok && val != "" {
			return val
		}
		return defaultVal
	})
}

// ConsumerConfig is one domain-scoped consumer process.
type ConsumerConfig struct {
	Host    string
	Port    int
	GroupID string
	Events  []string
}

// Config holds runtime settings for taskEvents services.
type Config struct {
	Transport        domain.TransportKind
	BootstrapServers string
	InternalSecret   string
	RedisHost        string
	RedisPort        int
	RedisDB          int
	StreamKey        string
	Accounts         ConsumerConfig
	Projects         ConsumerConfig
	Cloud            ConsumerConfig
	Realtime         ConsumerConfig
	Billing          ConsumerConfig
	Notifications    ConsumerConfig
}

type consumerBlock struct {
	Host    string   `json:"host"`
	Port    int      `json:"port"`
	GroupID string   `json:"groupId"`
	Events  []string `json:"events"`
}

func defaultConsumers() Config {
	return Config{
		Transport:        domain.TransportKafka,
		BootstrapServers: "localhost:9093",
		RedisHost:        "127.0.0.1",
		RedisPort:        6379,
		StreamKey:        "domain-events:all",
		Accounts: ConsumerConfig{
			Host: "0.0.0.0", Port: 8005, GroupID: "task-events-accounts",
			Events: []string{"USER_CREATED", "COMPANY_CREATED", "USER_ACTIVATED", "USER_LOGGED_IN"},
		},
		Projects: ConsumerConfig{
			Host: "0.0.0.0", Port: 8006, GroupID: "task-events-projects",
			Events: []string{"WORKSPACE_CREATED", "PROJECT_UPDATED", "TASK_COMPLETED", "AI_ASSISTANT_REPLY_COMPLETED"},
		},
		Cloud: ConsumerConfig{
			Host: "0.0.0.0", Port: 8007, GroupID: "task-events-cloud",
			Events: []string{"CLOUD_SERVER_STARTED", "CLOUD_SERVER_STOPPED", "CLOUD_SERVER_START_AUTO", "CLOUD_PLATFORM_AUTHORIZATION_CREATED"},
		},
		Realtime: ConsumerConfig{
			Host: "0.0.0.0", Port: 8008, GroupID: "task-events-realtime",
			Events: []string{"SSE_MESSAGE"},
		},
		Billing: ConsumerConfig{
			Host: "0.0.0.0", Port: 8009, GroupID: "task-events-billing",
			Events: []string{"BILLING_TRANSACTION_CREATED"},
		},
		Notifications: ConsumerConfig{
			Host: "0.0.0.0", Port: 8012, GroupID: "task-events-notifications",
			Events: []string{"EMAIL_SENT", "INVITATION_CREATED", "USER_ACTIVATED"},
		},
	}
}

// Load reads conf/events/domain-events/config.yaml from monorepo root.
func Load() (Config, string, error) {
	root, err := FindMonorepoRoot()
	if err != nil {
		return Config{}, "", err
	}
	cfg := defaultConsumers()
	block, err := loadDomainEventsGlobal(root)
	if err != nil {
		return cfg, root, err
	}
	if t := domain.TransportKind(strings.TrimSpace(block.Transport)); t.Valid() {
		cfg.Transport = t
	}
	if block.Kafka.BootstrapServers != "" {
		cfg.BootstrapServers = resolveTemplate(block.Kafka.BootstrapServers)
	}
	if block.InternalSecret != "" {
		cfg.InternalSecret = resolveTemplate(block.InternalSecret)
	}
	if block.Redis.Host != "" {
		cfg.RedisHost = resolveTemplate(block.Redis.Host)
	}
	if block.Redis.Port != 0 {
		cfg.RedisPort = block.Redis.Port
	}
	cfg.RedisDB = block.Redis.DB
	prefix := strings.TrimRight(block.Redis.StreamKeyPrefix, ":")
	if prefix != "" {
		cfg.StreamKey = prefix + ":all"
	}
	return cfg, root, nil
}

func mergeConsumer(dst *ConsumerConfig, src consumerBlock) {
	if src.Host != "" {
		dst.Host = src.Host
	}
	if src.Port != 0 {
		dst.Port = src.Port
	}
	if src.GroupID != "" {
		dst.GroupID = src.GroupID
	}
	if len(src.Events) > 0 {
		dst.Events = src.Events
	}
}

// ConsumerByName returns config for a named domain process.
func (c Config) ConsumerByName(name string) (ConsumerConfig, bool) {
	switch strings.ToLower(name) {
	case "accounts":
		return c.Accounts, true
	case "projects":
		return c.Projects, true
	case "cloud":
		return c.Cloud, true
	case "realtime":
		return c.Realtime, true
	case "billing":
		return c.Billing, true
	case "notifications":
		return c.Notifications, true
	default:
		return ConsumerConfig{}, false
	}
}

// EventTopic maps event_type to Kafka topic (aligned with Django KAFKA_TOPICS).
func EventTopic(eventType string) (string, bool) {
	topics := map[string]string{
		"USER_CREATED":                         "user-created",
		"USER_ACTIVATED":                       "user-activated",
		"USER_LOGGED_IN":                       "user-logged-in",
		"PROJECT_UPDATED":                      "project-updated",
		"CLOUD_SERVER_STARTED":                 "cloud-server-started",
		"CLOUD_SERVER_START_AUTO":              "cloud-server-start-auto",
		"CLOUD_SERVER_STOPPED":                 "cloud-server-stopped",
		"TASK_COMPLETED":                       "task-completed",
		"TASK_STATUS_CHANGED":                  "task-status-changed",
		"TASK_CREATED":                         "task-created",
		"TASK_DELETED":                         "task-deleted",
		"TASK_GRACEFUL_SHUTDOWN_AWAIT":         "task-graceful-shutdown-await",
		"TASK_COMMENT_IMAGE_MENTIONED":         "task-comment-image-mentioned",
		"CONTAINER_MIGRATE_AWAIT_READY":        "container-migrate-await-ready",
		"WORKSPACE_MACHINE_IDLE":               "workspace-machine-idle",
		"WORKSPACE_CREATED":                    "workspace-created",
		"BILLING_TRANSACTION_CREATED":          "billing-transaction-created",
		"COMPANY_CREATED":                      "company-created",
		"CLOUD_PLATFORM_AUTHORIZATION_CREATED": "cloud-platform-authorization-created",
		"CommentContainerBindingAdvanced":      "comment-container-binding-advanced",
		"LayerGraphSnapshotPersisted":          "layer-graph-snapshot-persisted",
		"JobStepFullArchived":                  "job-step-full-archived",
		"CommentStartupLogArchived":            "comment-startup-log-archived",
		"StepFullCOSConfigUpdated":             "step-full-cos-config-updated",
		"CONTAINER_INSTRUCTION_IDLE_MARKED":    "container-instruction-idle-marked",
		"CONTAINER_INSTRUCTION_IDLE_CLEARED":   "container-instruction-idle-cleared",
		"SSE_MESSAGE":                          "sse-message",
		"AI_ASSISTANT_REPLY_COMPLETED":         "ai-assistant-reply-completed",
		"EMAIL_SENT":                           "email-sent",
		"EMAIL_UNSUBSCRIBED":                   "email-unsubscribed",
		"EMAIL_RESUBSCRIBED":                   "email-resubscribed",
		"INVITATION_CREATED":                   "invitation-created",
		"REGISTRATION_INVITE_POLICY_UPDATED":   "registration-invite-policy-updated",
		"REGISTRATION_INVITE_CODE_ISSUED":      "registration-invite-code-issued",
		"REGISTRATION_INVITE_CODE_REDEEMED":    "registration-invite-code-redeemed",
		"RELAY_REGISTER_ATTEMPTED":             "relay-lifecycle",
		"RELAY_REGISTER_SUCCEEDED":             "relay-lifecycle",
		"RELAY_REGISTER_FAILED":                "relay-lifecycle",
		"RELAY_START_ACCEPTED":                 "relay-lifecycle",
		"RELAY_START_ATTEMPTED":                "relay-lifecycle",
		"RELAY_START_DISPATCH_SUCCEEDED":       "relay-lifecycle",
		"RELAY_START_DISPATCH_FAILED":          "relay-lifecycle",
		"RELAY_STOP_REQUESTED":                 "relay-lifecycle",
		"RELAY_STOP_SUCCEEDED":                 "relay-lifecycle",
		"RELAY_STOP_FAILED":                    "relay-lifecycle",
		"FEATURE_PARAMS_INITIALIZED":           "feature-params-initialized",
		"WECHAT_IDENTITY_CONFLICT":             "wechat-identity-conflict",
		"MEMBER_JOINED":                        "member-joined",
		"RoleChanged":                          "role-changed",
		"PlatformRoleChanged":                  "platform-role-changed",
		"RoleResourceGroupsChanged":            "role-resource-groups-changed",
		"UserImpersonationStarted":             "user-impersonation-started",
		"UserImpersonationStopped":             "user-impersonation-stopped",
		"UserInboxMessageCreated":              "user-inbox-message-created",
		"TASK_REVISION_RECORDED":               "task-revision-recorded",
		"PROJECT_REVISION_RECORDED":            "project-revision-recorded",
		"PROJECT_DELETED":                      "project-deleted",
		// taskBill 发布：Aliyun GitLab 区域待人工建节点 → ops alert（OPT-20260902-014）
		"GITLAB_MANUAL_NODE_FULFILLMENT_QUEUED": "gitlab-manual-node-fulfillment-queued",
	}
	t, ok := topics[eventType]
	return t, ok
}

// TopicsForEvents resolves subscribed event types to Kafka topics.
func TopicsForEvents(events []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, e := range events {
		t, ok := EventTopic(e)
		if !ok {
			continue
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// DeadLetterTopic returns the dead-letter topic name for a given topic.
func DeadLetterTopic(topic string) string {
	return topic + "-dlt"
}

// DLTMaxRetries returns the maximum number of retry attempts before a
// retryable message is sent to the dead-letter topic.  Overridable via
// DLT_MAX_RETRIES environment variable; defaults to 10.
func DLTMaxRetries() int {
	if v := os.Getenv("DLT_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 10
}

// DispatchPath returns Django internal dispatch URL for a domain.
func DispatchPath(domain string) string {
	return "/api/internal/task-events/" + domain + "/dispatch/"
}
