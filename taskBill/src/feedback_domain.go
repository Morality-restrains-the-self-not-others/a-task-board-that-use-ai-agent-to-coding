package main

import "strings"

const (
	feedbackKindTaskPost       = "task_post"
	feedbackKindGitlabTraffic  = "gitlab_traffic"
	feedbackKindGitlabDisk     = "gitlab_disk"
	feedbackKindConsumedAmount = "consumed_amount"
	feedbackTenantRegionKey    = "nav.feedback.main"
)

type FeedbackThreshold struct {
	ResourceKind string
	MinQuantity  float64
}

type FeedbackLinkSpec struct {
	Title     string
	URL       string
	SortOrder int
	Enabled   bool
}

type FeedbackGroupSpec struct {
	Enabled    bool
	Thresholds []FeedbackThreshold
	Links      []FeedbackLinkSpec
}

// groupVisible reports whether a tenant snapshot meets every configured threshold (AND, >=).
// Empty thresholds are vacuously true. Unspecified kinds in the snapshot count as 0.
func groupVisible(thresholds []FeedbackThreshold, snapshot map[string]float64) bool {
	if len(thresholds) == 0 {
		return true
	}
	for _, t := range thresholds {
		qty := 0.0
		if snapshot != nil {
			qty = snapshot[t.ResourceKind]
		}
		if qty < t.MinQuantity {
			return false
		}
	}
	return true
}

func validateHttpsURL(raw string) error {
	u := strings.TrimSpace(raw)
	if u == "" {
		return errFeedbackInvalidURL
	}
	lower := strings.ToLower(u)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") {
		return errFeedbackInvalidURL
	}
	if !strings.HasPrefix(lower, "https://") {
		return errFeedbackInvalidURL
	}
	return nil
}
