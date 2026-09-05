package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestAIInstructChunkBatcherFlushesOnSize(t *testing.T) {
	orig := publishAIInstructStreamSSE
	var n atomic.Int32
	publishAIInstructStreamSSE = func(ctx context.Context, taskID, instructID, phase, message string, httpStatus *int, apiEndpoint, traceID string) {
		if phase == "chunk" {
			n.Add(1)
		}
	}
	t.Cleanup(func() { publishAIInstructStreamSSE = orig })

	b := newAIInstructChunkBatcher(context.Background(), "t1", "c1", "tr")
	// Force size flush: write past aiChunkBatchMaxLen in one AddChunk after partial fill.
	big := make([]byte, aiChunkBatchMaxLen)
	for i := range big {
		big[i] = 'a'
	}
	b.AddChunk(string(big))
	if n.Load() != 1 {
		t.Fatalf("expected 1 flush on size, got %d", n.Load())
	}
}

func TestAIInstructChunkBatcherFlushesOnTimer(t *testing.T) {
	orig := publishAIInstructStreamSSE
	var n atomic.Int32
	var last string
	publishAIInstructStreamSSE = func(ctx context.Context, taskID, instructID, phase, message string, httpStatus *int, apiEndpoint, traceID string) {
		if phase == "chunk" {
			n.Add(1)
			last = message
		}
	}
	t.Cleanup(func() { publishAIInstructStreamSSE = orig })

	b := newAIInstructChunkBatcher(context.Background(), "t1", "c1", "tr")
	b.AddChunk("hello")
	b.AddChunk(" world")
	time.Sleep(aiChunkBatchWindow + 40*time.Millisecond)
	if n.Load() != 1 {
		t.Fatalf("expected 1 timed flush, got %d", n.Load())
	}
	if last != "hello world" {
		t.Fatalf("merged=%q", last)
	}
}
