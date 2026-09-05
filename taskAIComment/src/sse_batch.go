package main

import (
	"context"
	"strings"
	"sync"
	"time"
)

const (
	aiChunkBatchWindow = 80 * time.Millisecond
	aiChunkBatchMaxLen = 2048
)

// aiInstructChunkBatcher coalesces rapid "chunk" SSE publishes to reduce
// O(chunks) HTTP fan-out to taskSSE. error/done bypass and flush immediately.
type aiInstructChunkBatcher struct {
	mu        sync.Mutex
	buf       strings.Builder
	timer     *time.Timer
	taskID    string
	instructID string
	traceID   string
	ctx       context.Context
}

func newAIInstructChunkBatcher(ctx context.Context, taskID, instructID, traceID string) *aiInstructChunkBatcher {
	return &aiInstructChunkBatcher{
		ctx:        ctx,
		taskID:     taskID,
		instructID: instructID,
		traceID:    traceID,
	}
}

func (b *aiInstructChunkBatcher) AddChunk(chunk string) {
	if chunk == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.WriteString(chunk)
	if b.buf.Len() >= aiChunkBatchMaxLen {
		b.flushLocked()
		return
	}
	if b.timer == nil {
		b.timer = time.AfterFunc(aiChunkBatchWindow, func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			b.flushLocked()
		})
	}
}

func (b *aiInstructChunkBatcher) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flushLocked()
}

func (b *aiInstructChunkBatcher) flushLocked() {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	if b.buf.Len() == 0 {
		return
	}
	msg := b.buf.String()
	b.buf.Reset()
	// Publish outside holding only the batch mutex; publish is itself bounded by cloudHTTP.
	publishAIInstructStreamSSE(b.ctx, b.taskID, b.instructID, "chunk", msg, nil, "", b.traceID)
}
