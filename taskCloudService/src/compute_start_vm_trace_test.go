package main

import (
	"context"
	"net/http/httptest"
	"testing"

	"tracelog"
)

func TestBindStartVmTraceContextDoesNotUseTaskID(t *testing.T) {
	taskID := "task_15742467311115154867"
	ctx1, a := bindStartVmTraceContext(context.Background(), nil, taskID)
	_, b := bindStartVmTraceContext(context.Background(), nil, taskID)
	if a == "" || b == "" {
		t.Fatalf("empty trace: a=%q b=%q", a, b)
	}
	if a == taskID || b == taskID {
		t.Fatalf("must not use task_id as trace: a=%q b=%q", a, b)
	}
	if a == b {
		t.Fatalf("two starts on the same task must have independent traces, got %q", a)
	}
	if got := tracelog.TraceIDFromContext(ctx1); got != a {
		t.Fatalf("ctx trace=%q want %q", got, a)
	}
}

func TestBindStartVmTraceContextKeepsInboundHeader(t *testing.T) {
	taskID := "task_15742467311115154867"
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set(tracelog.Header, "web-start-comment-aaa")
	_, got := bindStartVmTraceContext(context.Background(), req, taskID)
	if got != "web-start-comment-aaa" {
		t.Fatalf("got %q want inbound header", got)
	}
}

func TestBindStartVmTraceContextRejectsInboundTaskID(t *testing.T) {
	taskID := "task_15742467311115154867"
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set(tracelog.Header, taskID)
	_, got := bindStartVmTraceContext(context.Background(), req, taskID)
	if got == "" || got == taskID {
		t.Fatalf("inbound task_id must be replaced, got %q", got)
	}
}
