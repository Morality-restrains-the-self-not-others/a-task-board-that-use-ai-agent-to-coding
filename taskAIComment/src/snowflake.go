package main

import (
	"os"
	"strconv"
	"sync"
	"time"
)

// Generator mirrors task2app/Saas_project/core/utils/snowflake.py
type Generator struct {
	epoch          int64
	machineID      int64
	sequence       int64
	lastTimestamp  int64
	maxSequence    int64
	timestampShift int
	machineIDShift int
	mu             sync.Mutex
}

var defaultGen = NewGenerator(defaultMachineID())

func NewGenerator(machineID int64) *Generator {
	if machineID > (1<<10)-1 {
		machineID = machineID % (1 << 10)
	}
	return &Generator{
		epoch:          1577836800000,
		machineID:      machineID,
		maxSequence:    (1 << 12) - 1,
		timestampShift: 22,
		machineIDShift: 12,
	}
}

func defaultMachineID() int64 {
	if v := os.Getenv("MACHINE_ID"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n % (1 << 10)
		}
	}
	return 7
}

func (g *Generator) Generate() int64 {
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

// GenerateID returns a new snowflake id using the process-wide generator.
func GenerateID() int64 {
	return defaultGen.Generate()
}
