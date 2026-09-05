// Package snowflake provides a shared Snowflake ID generator.
// OPT-20260726-018: Extracted from taskEvents/internal/snowflake,
// taskTenantService/src/snowflake.go, taskCloudService/src/snowflake.go
// to eliminate duplicate implementations across Go services.
//
// All services should import "snowflake" and use either:
//   - GenerateID() int64       — raw int64
//   - GenerateIDString() string — string representation
package snowflake

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

// Generator mirrors task2app/Saas_project/core/utils/snowflake.py
// epoch=2020-01-01, 41+10+12 bit layout
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

// NewGenerator creates a new Snowflake generator with the given machine ID.
func NewGenerator(machineID int64) *Generator {
	if machineID > (1<<10)-1 {
		machineID = machineID % (1 << 10)
	}
	return &Generator{
		epoch:           1577836800000, // 2020-01-01T00:00:00Z
		machineID:       machineID,
		maxSequence:     (1 << 12) - 1,
		timestampShift:  22,
		machineIDShift:  12,
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

// Generate returns a new Snowflake ID as int64.
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

// GenerateID returns a new Snowflake ID using the process-wide generator.
func GenerateID() int64 {
	return defaultGen.Generate()
}

// GenerateIDString returns a new Snowflake ID as a string, for services
// that prefer string IDs (matches the previous GenerateSnowflakeID convention).
func GenerateIDString() string {
	return fmt.Sprintf("%d", GenerateID())
}
