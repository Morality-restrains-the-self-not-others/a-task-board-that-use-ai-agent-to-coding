package tracelog

import (
	"log/slog"
	"strings"
)

var (
	aidevServiceID string
	aidevTags      []string
)

// SetDaydaymoneyMeta configures daydaymoney_service_id / daydaymoney_tags for all structured log output.
// When tags is empty and serviceID is non-empty, defaults to []string{"svc:" + serviceID}.
func SetDaydaymoneyMeta(serviceID string, tags []string) {
	aidevServiceID = strings.TrimSpace(serviceID)
	aidevTags = nil
	for _, tag := range tags {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			aidevTags = append(aidevTags, trimmed)
		}
	}
	if len(aidevTags) == 0 && aidevServiceID != "" {
		aidevTags = []string{"svc:" + aidevServiceID}
	}
	applyAidevToDefaultLogger()
}

// AidevServiceID returns the configured daydaymoney service id (for tests).
func AidevServiceID() string {
	return aidevServiceID
}

// DaydaymoneyTags returns a copy of configured daydaymoney tags (for tests).
func DaydaymoneyTags() []string {
	if len(aidevTags) == 0 {
		return nil
	}
	out := make([]string, len(aidevTags))
	copy(out, aidevTags)
	return out
}

func aidevTagsJoined() string {
	if len(aidevTags) == 0 {
		return ""
	}
	return strings.Join(aidevTags, ",")
}

func appendAidevFields(payload map[string]string) {
	if id := AidevServiceID(); id != "" {
		payload["daydaymoney_service_id"] = id
	}
	if tags := aidevTagsJoined(); tags != "" {
		payload["daydaymoney_tags"] = tags
	}
}

func applyAidevToDefaultLogger() {
	if baseJSONHandler == nil {
		return
	}
	logger := slog.New(baseJSONHandler)
	if id := AidevServiceID(); id != "" {
		logger = logger.With("daydaymoney_service_id", id)
	}
	if tags := aidevTagsJoined(); tags != "" {
		logger = logger.With("daydaymoney_tags", tags)
	}
	slog.SetDefault(logger)
}
