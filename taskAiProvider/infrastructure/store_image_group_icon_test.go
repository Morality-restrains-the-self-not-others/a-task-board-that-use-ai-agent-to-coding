package infrastructure

import "testing"

func TestImageGroupIconURLEmptyKey(t *testing.T) {
	if got := ImageGroupIconURL(1, "  "); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestImageGroupIconURLConventionPathAndCacheBust(t *testing.T) {
	a := ImageGroupIconURL(42, "7/image_group_icon_9.png")
	b := ImageGroupIconURL(42, "7/image_group_icon_9.png")
	c := ImageGroupIconURL(42, "7/image_group_icon_10.png")
	if a != b {
		t.Fatalf("same key should be stable: %s vs %s", a, b)
	}
	if a == "" || a == c {
		t.Fatalf("a=%s c=%s", a, c)
	}
	if want := "/api/ai-provider/public-image-groups/42/icon?h="; !containsPrefix(a, want) {
		t.Fatalf("url=%s want prefix %s", a, want)
	}
}

func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
