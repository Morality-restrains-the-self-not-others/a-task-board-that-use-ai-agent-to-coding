package main

import (
	"strings"
	"sync"
	"time"
)

const (
	agentChunkBatchWindow = 80 * time.Millisecond
	agentChunkBatchMaxLen = 2048
)

// containerAgentAppendBatcher debounces DB writes for container agent stream chunks.
type containerAgentAppendBatcher struct {
	mu    sync.Mutex
	id    string
	total strings.Builder
	buf   strings.Builder
	timer *time.Timer
}

var agentAppendBatchers sync.Map

func getOrCreateAgentBatcher(id string) (*containerAgentAppendBatcher, error) {
	if v, ok := agentAppendBatchers.Load(id); ok {
		return v.(*containerAgentAppendBatcher), nil
	}
	b := &containerAgentAppendBatcher{id: id}
	if v, loaded := agentAppendBatchers.LoadOrStore(id, b); loaded {
		return v.(*containerAgentAppendBatcher), nil
	}
	c, err := loadContainerAgentComment(id)
	if err != nil {
		agentAppendBatchers.Delete(id)
		return nil, err
	}
	if c.AssistantResponse.Valid {
		b.total.WriteString(c.AssistantResponse.String)
	}
	return b, nil
}

func appendContainerAgentChunkBatched(id, chunk string) (merged string, updated bool, err error) {
	if chunk == "" {
		c, loadErr := loadContainerAgentComment(id)
		if loadErr != nil {
			return "", false, loadErr
		}
		text := ""
		if c.AssistantResponse.Valid {
			text = c.AssistantResponse.String
		}
		return text, true, nil
	}
	b, err := getOrCreateAgentBatcher(id)
	if err != nil {
		return "", false, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.total.WriteString(chunk)
	b.buf.WriteString(chunk)
	if b.buf.Len() >= agentChunkBatchMaxLen {
		if flushErr := b.flushLocked(); flushErr != nil {
			return "", false, flushErr
		}
	} else if b.timer == nil {
		b.timer = time.AfterFunc(agentChunkBatchWindow, func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			_ = b.flushLocked()
		})
	}
	return b.total.String(), true, nil
}

func flushContainerAgentBatcher(id string) error {
	v, ok := agentAppendBatchers.Load(id)
	if !ok {
		return nil
	}
	b := v.(*containerAgentAppendBatcher)
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushLocked()
}

func removeContainerAgentBatcher(id string) {
	agentAppendBatchers.Delete(id)
}

func (b *containerAgentAppendBatcher) flushLocked() error {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	if b.buf.Len() == 0 {
		return nil
	}
	b.buf.Reset()
	text := b.total.String()
	_, err := setContainerAgentAssistantResponse(b.id, text)
	return err
}

func (b *containerAgentAppendBatcher) Flush() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushLocked()
}
