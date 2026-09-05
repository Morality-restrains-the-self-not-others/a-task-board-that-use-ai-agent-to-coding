package broker

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"taskEvents/domain"
	"tracelog"
)

func TestBuildDLTMessageIncludesTraceIDFromContext(t *testing.T) {
	ctx := tracelog.ContextWithTraceID(context.Background(), "7bb8e633-22d5-426a-98d0-3be19422cfb4")
	env := domain.EventEnvelope{
		EventType: "TASK_COMMENT_IMAGE_MENTIONED",
		Key:       "task_1",
		Data:      json.RawMessage(`{"task_id":"task_1","trace_id":"7bb8e633-22d5-426a-98d0-3be19422cfb4"}`),
	}
	msg := buildDLTMessage(ctx, env, errors.New("boom"), FailureReasonPermanent, 1)
	if msg.TraceID != "7bb8e633-22d5-426a-98d0-3be19422cfb4" {
		t.Fatalf("TraceID=%q, want from context", msg.TraceID)
	}
	if msg.OtelTraceID == "" {
		t.Fatal("OtelTraceID should be populated")
	}
	if msg.OriginalEventType != "TASK_COMMENT_IMAGE_MENTIONED" {
		t.Fatalf("OriginalEventType=%q", msg.OriginalEventType)
	}
}

func TestBuildDLTMessageFallsBackToEnvelopeTraceID(t *testing.T) {
	env := domain.EventEnvelope{
		EventType: "USER_CREATED",
		Key:       "u1",
		Data:      json.RawMessage(`{"trace_id":"aabbccdd-1122-3344-5566-77889900aabb","user_id":"u1"}`),
	}
	msg := buildDLTMessage(context.Background(), env, errors.New("x"), FailureReasonRetryExhausted, 11)
	if msg.TraceID != "aabbccdd-1122-3344-5566-77889900aabb" {
		t.Fatalf("TraceID=%q, want envelope fallback", msg.TraceID)
	}
	if msg.RetryCount != 11 || msg.FailureReason != FailureReasonRetryExhausted {
		t.Fatalf("msg=%+v", msg)
	}
}

func TestFormatDLTAlertBodyIncludesTraceID(t *testing.T) {
	msg := sampleDLT()
	msg.TraceID = "7bb8e633-22d5-426a-98d0-3be19422cfb4"
	body := formatDLTAlertBody(msg, DefaultDLTAlertCooldown, 0)
	if !strings.Contains(body, "trace_id: 7bb8e633-22d5-426a-98d0-3be19422cfb4") {
		t.Fatalf("alert body missing trace_id:\n%s", body)
	}
}
