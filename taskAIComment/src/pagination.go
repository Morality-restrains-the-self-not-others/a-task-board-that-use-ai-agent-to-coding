package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultCommentLimit   = 50
	maxCommentLimit       = 200
	assistantPreviewLen   = 2048
)

type commentPageParams struct {
	limit           int
	cursorCreatedAt time.Time
	cursorID        string
	hasCursor       bool
	paginate        bool
	preview         bool
	full            bool
}

func parseCommentPageQuery(r *http.Request) commentPageParams {
	q := r.URL.Query()
	p := commentPageParams{
		limit:   defaultCommentLimit,
		preview: true,
	}
	if v := strings.TrimSpace(q.Get("preview")); v == "0" || strings.EqualFold(v, "false") {
		p.preview = false
	}
	if v := strings.TrimSpace(q.Get("full")); v == "1" || strings.EqualFold(v, "true") {
		p.full = true
		p.preview = false
	}
	if lim := strings.TrimSpace(q.Get("limit")); lim != "" {
		p.paginate = true
		n, err := strconv.Atoi(lim)
		if err != nil || n < 1 {
			n = defaultCommentLimit
		}
		if n > maxCommentLimit {
			n = maxCommentLimit
		}
		p.limit = n
	}
	if cur := strings.TrimSpace(q.Get("cursor")); cur != "" {
		if ts, id, ok := decodeCommentCursor(cur); ok {
			p.hasCursor = true
			p.cursorCreatedAt = ts
			p.cursorID = id
		}
	}
	return p
}

func encodeCommentCursor(createdAt time.Time, id string) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + id
	return url.QueryEscape(raw)
}

func decodeCommentCursor(raw string) (time.Time, string, bool) {
	decoded, err := url.QueryUnescape(strings.TrimSpace(raw))
	if err != nil {
		decoded = strings.TrimSpace(raw)
	}
	parts := strings.SplitN(decoded, "|", 2)
	if len(parts) != 2 || parts[1] == "" {
		return time.Time{}, "", false
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		ts, err = time.Parse(time.RFC3339, parts[0])
		if err != nil {
			return time.Time{}, "", false
		}
	}
	return ts, parts[1], true
}

func applyAssistantPreview(item map[string]interface{}, preview bool, full bool) {
	if full || !preview {
		return
	}
	var text string
	if v, ok := item["assistant_response"].(string); ok {
		text = v
	}
	if text == "" {
		return
	}
	item["assistant_len"] = len(text)
	if len(text) <= assistantPreviewLen {
		item["assistant_preview"] = text
		return
	}
	item["assistant_preview"] = text[:assistantPreviewLen]
	delete(item, "assistant_response")
}

func commentPageResponse(results []map[string]interface{}, hasMore bool, lastCreatedAt time.Time, lastID string) map[string]interface{} {
	out := map[string]interface{}{
		"results":  results,
		"has_more": hasMore,
	}
	if hasMore && !lastCreatedAt.IsZero() && lastID != "" {
		out["next_cursor"] = encodeCommentCursor(lastCreatedAt, lastID)
	} else {
		out["next_cursor"] = nil
	}
	return out
}
