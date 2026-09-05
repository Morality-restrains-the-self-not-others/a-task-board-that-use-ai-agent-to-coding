package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// StartAllProgressEvent represents a progress update during start-all / stop-all / build-all.
type StartAllProgressEvent struct {
	Total     int      `json:"total"`
	Started   int      `json:"started"`
	Skipped   int      `json:"skipped,omitempty"`
	Failed    int      `json:"failed"`
	Current   string   `json:"current"`
	Remaining int      `json:"remaining"`
	Done      bool     `json:"done"`
	Error     string   `json:"error,omitempty"`     // latest error (backward compat)
	Errors    []string `json:"errors,omitempty"`    // all accumulated errors
	Phase     string   `json:"phase"`               // "starting", "progress", "done", "error", "cancelled"
	Operation string   `json:"operation,omitempty"` // "start", "stop", or "build"
	// RunID is the bulk/single lifecycle run this event belongs to, stamped by
	// Publish so the SSE consumer can correlate an error banner with runAll logs.
	RunID string `json:"run_id,omitempty"`
	// Detail carries the full operation result payload when a task is not
	// one of the service lifecycle ops (e.g. dev database clear/init result
	// with per-step migrations/inits). Only populated on the final event.
	Detail any `json:"detail,omitempty"`
}

func progressPhaseForContextErr(ctx context.Context) string {
	if errors.Is(ctx.Err(), context.Canceled) {
		return "cancelled"
	}
	return "error"
}

// ToSSE formats the event as an SSE data line.
func (e StartAllProgressEvent) ToSSE() string {
	b, _ := json.Marshal(e)
	return fmt.Sprintf("data: %s\n\n", string(b))
}

// ProgressBroadcaster fans out progress events to all subscribers and keeps
// the latest event per runID so late subscribers (e.g. after page refresh)
// can catch up immediately.
type ProgressBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan StartAllProgressEvent]struct{} // runID -> set of channels
	latest      map[string]StartAllProgressEvent                   // runID -> last published event
}

// NewProgressBroadcaster creates a new ProgressBroadcaster.
func NewProgressBroadcaster() *ProgressBroadcaster {
	return &ProgressBroadcaster{
		subscribers: make(map[string]map[chan StartAllProgressEvent]struct{}),
		latest:      make(map[string]StartAllProgressEvent),
	}
}

// Subscribe returns a channel that receives progress events for the given runID.
// If a latest snapshot exists, it is delivered immediately (best-effort).
// The caller must call Unsubscribe when done.
func (b *ProgressBroadcaster) Subscribe(runID string) chan StartAllProgressEvent {
	ch := make(chan StartAllProgressEvent, 64)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subscribers[runID] == nil {
		b.subscribers[runID] = make(map[chan StartAllProgressEvent]struct{})
	}
	b.subscribers[runID][ch] = struct{}{}
	if ev, ok := b.latest[runID]; ok {
		select {
		case ch <- ev:
		default:
		}
	}
	return ch
}

// Latest returns the most recent event for runID, if any.
func (b *ProgressBroadcaster) Latest(runID string) (StartAllProgressEvent, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	ev, ok := b.latest[runID]
	return ev, ok
}

// Unsubscribe removes a subscriber channel. Safe to call multiple times.
func (b *ProgressBroadcaster) Unsubscribe(runID string, ch chan StartAllProgressEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if subs, ok := b.subscribers[runID]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(b.subscribers, runID)
		}
	}
}

// Publish sends an event to all subscribers of the given runID and stores it
// as the latest snapshot for late subscribers.
// Non-blocking: slow subscribers may miss events (buffer of 64).
func (b *ProgressBroadcaster) Publish(runID string, event StartAllProgressEvent) {
	if event.RunID == "" {
		event.RunID = runID
	}
	b.mu.Lock()
	b.latest[runID] = event
	subs := b.subscribers[runID]
	b.mu.Unlock()
	for ch := range subs {
		select {
		case ch <- event:
		default:
			// drop event for slow subscriber (buffer full)
		}
	}
}

// CloseRun closes and removes all subscribers for a runID and clears its snapshot.
func (b *ProgressBroadcaster) CloseRun(runID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subscribers[runID] {
		close(ch)
	}
	delete(b.subscribers, runID)
	delete(b.latest, runID)
}
