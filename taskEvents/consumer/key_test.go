package consumer

import (
	"encoding/json"
	"testing"

	"taskEvents/domain"
)

func TestIdempotencyKeyFromEnvelopePrefersEventID(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"task_id":  "task_1",
		"event_id": "862552507173216256",
	})
	env := domain.EventEnvelope{
		EventType: "CLOUD_SERVER_START_AUTO",
		Data:      data,
	}
	key := IdempotencyKeyFromEnvelope(env)
	want := domain.IdempotencyKeyForEvent("CLOUD_SERVER_START_AUTO", "862552507173216256")
	if key != want {
		t.Fatalf("expected %q, got %q", want, key)
	}
}

func TestIdempotencyKeyFromEnvelopeFallsBackToTaskID(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "task_1",
	})
	env := domain.EventEnvelope{
		EventType: "CLOUD_SERVER_START_AUTO",
		Data:      data,
	}
	key := IdempotencyKeyFromEnvelope(env)
	want := domain.IdempotencyKeyForEvent("CLOUD_SERVER_START_AUTO", "task_1")
	if key != want {
		t.Fatalf("expected %q, got %q", want, key)
	}
}

func TestIdempotencyKeyFromEnvelopeCloudServerStartIgnoresCompanyID(t *testing.T) {
	company := "850256677331562496"
	makeEnv := func(eventType, eventID, taskID, traceID string) domain.EventEnvelope {
		payload := map[string]interface{}{
			"task_id":    taskID,
			"company_id": company,
		}
		if eventID != "" {
			payload["event_id"] = eventID
		}
		if traceID != "" {
			payload["trace_id"] = traceID
		}
		data, _ := json.Marshal(payload)
		return domain.EventEnvelope{EventType: eventType, Data: data}
	}

	kStarted := IdempotencyKeyFromEnvelope(makeEnv("CLOUD_SERVER_STARTED", "evt-a", "task_a", ""))
	kAuto := IdempotencyKeyFromEnvelope(makeEnv("CLOUD_SERVER_START_AUTO", "evt-b", "task_b", ""))
	if kStarted == kAuto {
		t.Fatalf("different start event_ids must not share key: %q", kStarted)
	}
	wantStarted := domain.IdempotencyKeyForEvent("CLOUD_SERVER_STARTED", "evt-a")
	if kStarted != wantStarted {
		t.Fatalf("expected event_id key %q, got %q", wantStarted, kStarted)
	}

	// Same company + different tasks without event_id must not collapse to company_id.
	k1 := IdempotencyKeyFromEnvelope(makeEnv("CLOUD_SERVER_STARTED", "", "task_1", "trace-1"))
	k2 := IdempotencyKeyFromEnvelope(makeEnv("CLOUD_SERVER_STARTED", "", "task_2", "trace-2"))
	if k1 == k2 {
		t.Fatalf("missing event_id must not key by company_id: %q", k1)
	}
	want1 := domain.IdempotencyKeyForEvent("CLOUD_SERVER_STARTED", "task_1:trace-1")
	if k1 != want1 {
		t.Fatalf("expected task:trace fallback %q, got %q", want1, k1)
	}
}

func TestIdempotencyKeyFromEnvelopeCloudServerStoppedIgnoresCompanyID(t *testing.T) {
	makeEnv := func(taskID, instanceID, stopRequestID string) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"task_id":          taskID,
			"company_id":       "850256677331562496",
			"instance_id":      instanceID,
			"stop_request_id":  stopRequestID,
			"authorization_id": "862031128628060160",
		})
		return domain.EventEnvelope{EventType: "CLOUD_SERVER_STOPPED", Data: data}
	}
	k1 := IdempotencyKeyFromEnvelope(makeEnv("task_a", "mock-e2e-1", "req-1"))
	k2 := IdempotencyKeyFromEnvelope(makeEnv("task_b", "mock-e2e-2", "req-2"))
	if k1 == k2 {
		t.Fatalf("different stop requests must not share idempotency key: %q", k1)
	}
	want1 := domain.IdempotencyKeyForEvent("CLOUD_SERVER_STOPPED", "req-1")
	if k1 != want1 {
		t.Fatalf("expected stop_request_id key %q, got %q", want1, k1)
	}
	// Without stop_request_id, fall back to task_id:instance_id (still not company_id).
	data, _ := json.Marshal(map[string]interface{}{
		"task_id":     "task_c",
		"company_id":  "850256677331562496",
		"instance_id": "mock-e2e-1",
	})
	k3 := IdempotencyKeyFromEnvelope(domain.EventEnvelope{EventType: "CLOUD_SERVER_STOPPED", Data: data})
	want3 := domain.IdempotencyKeyForEvent("CLOUD_SERVER_STOPPED", "task_c:mock-e2e-1")
	if k3 != want3 {
		t.Fatalf("expected task:instance key %q, got %q", want3, k3)
	}
}

func TestIdempotencyKeyFromEnvelopeTaskStatusChangedUsesStatusFingerprint(t *testing.T) {
	makeEnv := func(columnID, prevColumnID string, completed, prevCompleted bool, name string) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"task_id":                     "task_13282139435972172871",
			"company_id":                  "850256677331562496",
			"progress_column_id":          columnID,
			"previous_progress_column_id": prevColumnID,
			"completed":                   completed,
			"previous_completed":          prevCompleted,
			"progress_column_name":        name,
		})
		return domain.EventEnvelope{EventType: "TASK_STATUS_CHANGED", Data: data}
	}
	kCancel := IdempotencyKeyFromEnvelope(makeEnv("col-cancel", "col-wip", false, false, "已取消"))
	kDone := IdempotencyKeyFromEnvelope(makeEnv("col-done", "col-wip", true, false, "已完成"))
	if kCancel == kDone {
		t.Fatalf("distinct terminal transitions must not share key: %q", kCancel)
	}
	kDup := IdempotencyKeyFromEnvelope(makeEnv("col-cancel", "col-wip", false, false, "已取消"))
	if kCancel != kDup {
		t.Fatalf("identical status change should dedupe: %q vs %q", kCancel, kDup)
	}
	// company_id alone must not collapse keys across tasks
	dataOther, _ := json.Marshal(map[string]interface{}{
		"task_id":                     "task_other",
		"company_id":                  "850256677331562496",
		"progress_column_id":          "col-cancel",
		"previous_progress_column_id": "col-wip",
		"completed":                   false,
		"previous_completed":          false,
		"progress_column_name":        "已取消",
	})
	kOther := IdempotencyKeyFromEnvelope(domain.EventEnvelope{EventType: "TASK_STATUS_CHANGED", Data: dataOther})
	if kOther == kCancel {
		t.Fatalf("different tasks must not share key: %q", kCancel)
	}
}

func TestIdempotencyKeyFromEnvelopeTaskCreatedIgnoresUserID(t *testing.T) {
	userID := "850256676127797248"
	makeEnv := func(taskID, title string) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"task_id":      taskID,
			"tenant_id":    "850256677331562496",
			"company_id":   "850256677331562496",
			"workspace_id": "857903329669984256",
			"title":        title,
			"user_id":      userID,
		})
		return domain.EventEnvelope{EventType: "TASK_CREATED", Data: data}
	}
	k1 := IdempotencyKeyFromEnvelope(makeEnv("task_a", "fork A"))
	k2 := IdempotencyKeyFromEnvelope(makeEnv("task_b", "fork B"))
	if k1 == k2 {
		t.Fatalf("distinct TASK_CREATED must not collapse on user_id: %q", k1)
	}
	want1 := domain.IdempotencyKeyForEvent("TASK_CREATED", "task_a")
	if k1 != want1 {
		t.Fatalf("expected task_id key %q, got %q", want1, k1)
	}
	kDup := IdempotencyKeyFromEnvelope(makeEnv("task_a", "fork A again"))
	if k1 != kDup {
		t.Fatalf("identical task_id should dedupe: %q vs %q", k1, kDup)
	}
}

func TestIdempotencyKeyFromEnvelopeTaskDeletedIgnoresUserID(t *testing.T) {
	makeEnv := func(taskID string) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"task_id":      taskID,
			"tenant_id":    "850256677331562496",
			"company_id":   "850256677331562496",
			"workspace_id": "857903329669984256",
			"user_id":      "850256676127797248",
		})
		return domain.EventEnvelope{EventType: "TASK_DELETED", Data: data}
	}
	k1 := IdempotencyKeyFromEnvelope(makeEnv("task_del_1"))
	k2 := IdempotencyKeyFromEnvelope(makeEnv("task_del_2"))
	if k1 == k2 {
		t.Fatalf("distinct TASK_DELETED must not collapse on user_id: %q", k1)
	}
	want1 := domain.IdempotencyKeyForEvent("TASK_DELETED", "task_del_1")
	if k1 != want1 {
		t.Fatalf("expected task_id key %q, got %q", want1, k1)
	}
}

func TestIdempotencyKeyFromEnvelopeSSEMessageDistinctByStatusData(t *testing.T) {
	makeEnv := func(progress int) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"task_id": "task_1",
			"status_data": map[string]interface{}{
				"status":   "processing",
				"progress": progress,
				"message":  "step",
			},
		})
		return domain.EventEnvelope{EventType: "SSE_MESSAGE", Data: data}
	}
	k1 := IdempotencyKeyFromEnvelope(makeEnv(35))
	k2 := IdempotencyKeyFromEnvelope(makeEnv(45))
	if k1 == k2 {
		t.Fatalf("SSE progress updates must not share idempotency key: %q", k1)
	}
	kDup := IdempotencyKeyFromEnvelope(makeEnv(35))
	if k1 != kDup {
		t.Fatalf("identical SSE payload should dedupe: %q vs %q", k1, kDup)
	}
}

func TestIdempotencyKeyFromEnvelopeCloudPlatformAuthorizationUsesAuthID(t *testing.T) {
	makeEnv := func(authID, companyID string) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"id":            authID,
			"company_id":    companyID,
			"platform_type": "aliyun",
		})
		return domain.EventEnvelope{EventType: "CLOUD_PLATFORM_AUTHORIZATION_CREATED", Data: data}
	}
	k1 := IdempotencyKeyFromEnvelope(makeEnv("auth-1", "850256677331562496"))
	k2 := IdempotencyKeyFromEnvelope(makeEnv("auth-2", "850256677331562496"))
	if k1 == k2 {
		t.Fatalf("same-tenant authorizations must not collapse to one key: %q", k1)
	}
	want := domain.IdempotencyKeyForEvent("CLOUD_PLATFORM_AUTHORIZATION_CREATED", "auth-1")
	if k1 != want {
		t.Fatalf("expected auth id key %q, got %q", want, k1)
	}
}

func TestIdempotencyKeyFromEnvelopeBillingOrderCommentKeysByCommentID(t *testing.T) {
	makeEnv := func(commentID, orderID string) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"comment_id":     commentID,
			"order_id":       orderID,
			"tenant_id":      "t1",
			"author_user_id": "u1",
		})
		return domain.EventEnvelope{EventType: "BILLING_ORDER_COMMENT_CREATED", Data: data}
	}
	k1 := IdempotencyKeyFromEnvelope(makeEnv("cmt_1", "ORD-1"))
	k2 := IdempotencyKeyFromEnvelope(makeEnv("cmt_2", "ORD-1"))
	if k1 == k2 {
		t.Fatalf("different comments on the same order must not share key: %q", k1)
	}
	want := domain.IdempotencyKeyForEvent("BILLING_ORDER_COMMENT_CREATED", "cmt_1")
	if k1 != want {
		t.Fatalf("expected comment_id key %q, got %q", want, k1)
	}
}

func TestIdempotencyKeyFromEnvelopeTaskCommentImageMentionedKeysByCommentID(t *testing.T) {
	makeEnv := func(commentID, taskID string) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"task_id":            taskID,
			"tenant_id":          "t1",
			"company_id":         "t1",
			"comment_id":         commentID,
			"parent_comment_id":  commentID,
			"installed_image_id": "img-1",
			"created_by_id":      "u1",
		})
		return domain.EventEnvelope{EventType: "TASK_COMMENT_IMAGE_MENTIONED", Data: data}
	}
	k1 := IdempotencyKeyFromEnvelope(makeEnv("cmt_878866202137489408", "task_878865840278106112"))
	k2 := IdempotencyKeyFromEnvelope(makeEnv("cmt_878881789211340800", "task_878865840278106112"))
	if k1 == k2 {
		t.Fatalf("different @mention comments on the same task must not share key: %q", k1)
	}
	want := domain.IdempotencyKeyForEvent("TASK_COMMENT_IMAGE_MENTIONED", "cmt_878866202137489408")
	if k1 != want {
		t.Fatalf("expected comment_id key %q, got %q", want, k1)
	}
	kDup := IdempotencyKeyFromEnvelope(makeEnv("cmt_878866202137489408", "task_878865840278106112"))
	if k1 != kDup {
		t.Fatalf("replay of the same comment must keep the key: %q vs %q", k1, kDup)
	}
}

func TestIdempotencyKeyFromEnvelopeRevisionRecordedKeysByRevisionID(t *testing.T) {
	makeEnv := func(eventType, rev, entity string) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"revision_id": rev,
			"task_id":     entity,
			"project_id":  entity,
			"tenant_id":   "t1",
			"user_id":     "u1",
		})
		return domain.EventEnvelope{EventType: eventType, Data: data, Key: entity}
	}
	k1 := IdempotencyKeyFromEnvelope(makeEnv("TASK_REVISION_RECORDED", "rev-1", "task_a"))
	k2 := IdempotencyKeyFromEnvelope(makeEnv("TASK_REVISION_RECORDED", "rev-2", "task_a"))
	if k1 == k2 {
		t.Fatalf("different revision_id on the same task must not share key: %q", k1)
	}
	want := domain.IdempotencyKeyForEvent("TASK_REVISION_RECORDED", "rev-1")
	if k1 != want {
		t.Fatalf("expected revision_id key %q, got %q", want, k1)
	}
	kP1 := IdempotencyKeyFromEnvelope(makeEnv("PROJECT_REVISION_RECORDED", "prev-1", "proj_a"))
	kP2 := IdempotencyKeyFromEnvelope(makeEnv("PROJECT_REVISION_RECORDED", "prev-2", "proj_a"))
	if kP1 == kP2 {
		t.Fatalf("different project revision_id must not share key: %q", kP1)
	}
}

func TestIdempotencyKeyFromEnvelopeProjectDeletedUsesProjectID(t *testing.T) {
	makeEnv := func(projectID, tenantID string) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"project_id": projectID,
			"tenant_id":  tenantID,
		})
		return domain.EventEnvelope{EventType: "PROJECT_DELETED", Data: data, Key: projectID}
	}
	k1 := IdempotencyKeyFromEnvelope(makeEnv("proj_a", "tenant-same"))
	k2 := IdempotencyKeyFromEnvelope(makeEnv("proj_b", "tenant-same"))
	if k1 == k2 {
		t.Fatalf("different project_id must not share key: %q", k1)
	}
	want := domain.IdempotencyKeyForEvent("PROJECT_DELETED", "proj_a")
	if k1 != want {
		t.Fatalf("expected project_id key %q, got %q", want, k1)
	}
}

func TestIdempotencyKeyFromEnvelopeGitlabManualNodeUsesOrderID(t *testing.T) {
	makeEnv := func(orderID, tenantID string) domain.EventEnvelope {
		data, _ := json.Marshal(map[string]interface{}{
			"tenant_id":      tenantID,
			"order_id":       orderID,
			"region_slug":    "aliyun-cn-hangzhou",
			"cloud_provider": "aliyun",
		})
		return domain.EventEnvelope{EventType: "GITLAB_MANUAL_NODE_FULFILLMENT_QUEUED", Data: data, Key: orderID}
	}
	k1 := IdempotencyKeyFromEnvelope(makeEnv("9300000100", "9300000201"))
	k2 := IdempotencyKeyFromEnvelope(makeEnv("9300000101", "9300000201"))
	if k1 == k2 {
		t.Fatalf("different orders in the same tenant must not share key: %q", k1)
	}
	want := domain.IdempotencyKeyForEvent("GITLAB_MANUAL_NODE_FULFILLMENT_QUEUED", "9300000100")
	if k1 != want {
		t.Fatalf("expected order_id key %q, got %q", want, k1)
	}
	kDup := IdempotencyKeyFromEnvelope(makeEnv("9300000100", "9300000201"))
	if k1 != kDup {
		t.Fatalf("replay of the same order must keep the key: %q vs %q", k1, kDup)
	}
}
