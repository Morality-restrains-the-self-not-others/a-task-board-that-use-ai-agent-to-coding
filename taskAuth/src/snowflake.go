package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// SnowflakeGenerator mirrors task2app/Saas_project/core/utils/snowflake.py
type SnowflakeGenerator struct {
	epoch            int64
	machineID        int64
	sequence         int64
	lastTimestamp    int64
	maxSequence      int64
	timestampShift   int
	machineIDShift   int
	mu               sync.Mutex
}

var snowflake = &SnowflakeGenerator{
	epoch:          1577836800000,
	machineID:      7,
	maxSequence:    (1 << 12) - 1,
	timestampShift: 22,
	machineIDShift: 12,
}

func (g *SnowflakeGenerator) generate() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	ts := time.Now().UnixMilli()
	if ts < g.lastTimestamp {
		panic("clock moved backwards")
	}
	if ts == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & g.maxSequence
		if g.sequence == 0 {
			for ts <= g.lastTimestamp {
				ts = time.Now().UnixMilli()
			}
		}
	} else {
		g.sequence = 0
	}
	g.lastTimestamp = ts
	return ((ts - g.epoch) << g.timestampShift) | (g.machineID << g.machineIDShift) | g.sequence
}

func generateSnowflakeID() int64 {
	return snowflake.generate()
}

func generateTokenKey() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func generateActivationToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
