package saas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func taskAICommentBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("TASK_AI_COMMENT_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8019"
}

func taskAICommentInternalSecret() string {
	return strings.TrimSpace(os.Getenv("TASK_AI_COMMENT_INTERNAL_SECRET"))
}

var aiCommentHTTP = &http.Client{Timeout: 15 * time.Second}

// UpdateAssistantResponseViaHTTP PATCHes taskAIComment Go service (preferred over Django table).
func (r *Repository) UpdateAssistantResponseViaHTTP(commentID int64, text string) (bool, error) {
	url := fmt.Sprintf("%s/api/internal/task-ai-comment/%d/assistant-response/", taskAICommentBaseURL(), commentID)
	body, err := json.Marshal(map[string]string{"assistant_response": text})
	if err != nil {
		return false, err
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := taskAICommentInternalSecret(); sec != "" {
		req.Header.Set("X-TaskAIComment-Internal-Secret", sec)
	}
	resp, err := aiCommentHTTP.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return true, nil
	}
	return false, fmt.Errorf("taskAIComment PATCH status %d", resp.StatusCode)
}
