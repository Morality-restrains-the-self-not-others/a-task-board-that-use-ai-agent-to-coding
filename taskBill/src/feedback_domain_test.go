package main

import "testing"

func TestGroupVisibleEmptyThresholds(t *testing.T) {
	if !groupVisible(nil, nil) {
		t.Fatal("empty thresholds must be visible")
	}
	if !groupVisible([]FeedbackThreshold{}, map[string]float64{}) {
		t.Fatal("empty slice must be visible")
	}
}

func TestGroupVisibleTaskPostBoundary(t *testing.T) {
	th := []FeedbackThreshold{{ResourceKind: feedbackKindTaskPost, MinQuantity: 100}}
	if groupVisible(th, map[string]float64{feedbackKindTaskPost: 99}) {
		t.Fatal("99 must be hidden")
	}
	if !groupVisible(th, map[string]float64{feedbackKindTaskPost: 100}) {
		t.Fatal("100 must be visible")
	}
	if groupVisible(th, map[string]float64{}) {
		t.Fatal("missing kind treated as 0 must be hidden")
	}
}

func TestGroupVisibleAND(t *testing.T) {
	th := []FeedbackThreshold{
		{ResourceKind: feedbackKindTaskPost, MinQuantity: 100},
		{ResourceKind: feedbackKindGitlabTraffic, MinQuantity: 10},
	}
	if groupVisible(th, map[string]float64{feedbackKindTaskPost: 100, feedbackKindGitlabTraffic: 9}) {
		t.Fatal("AND fail must hide")
	}
	if !groupVisible(th, map[string]float64{feedbackKindTaskPost: 100, feedbackKindGitlabTraffic: 10}) {
		t.Fatal("AND pass must show")
	}
}

func TestGroupVisibleNested(t *testing.T) {
	low := []FeedbackThreshold{{ResourceKind: feedbackKindTaskPost, MinQuantity: 0}}
	high := []FeedbackThreshold{{ResourceKind: feedbackKindTaskPost, MinQuantity: 100}}
	snap := map[string]float64{feedbackKindTaskPost: 200}
	if !groupVisible(low, snap) || !groupVisible(high, snap) {
		t.Fatal("high consumption must see both groups")
	}
}

func TestValidateHttpsURL(t *testing.T) {
	if err := validateHttpsURL("https://example.com/a"); err != nil {
		t.Fatalf("https: %v", err)
	}
	for _, bad := range []string{"http://example.com", "javascript:alert(1)", "/rel", "ftp://x", ""} {
		if err := validateHttpsURL(bad); err == nil {
			t.Fatalf("want reject %q", bad)
		}
	}
}
