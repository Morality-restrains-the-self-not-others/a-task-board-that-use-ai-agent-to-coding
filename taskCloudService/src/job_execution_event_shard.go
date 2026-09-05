package main

import (
	"fmt"
	"hash/crc32"
	"regexp"
	"strings"
)

// ADR-0023 同款：cloud_job_execution_event 按 workspace_id 哈希分表。
// 表名 cloud_job_execution_event_{00..15}，选片 CRC32(workspace_id) % 16。
const jobExecutionEventShardCount = 16

var jobExecutionEventTableNameRe = regexp.MustCompile(`^cloud_job_execution_event_[0-9]{2}$`)

func jobExecutionEventShardIndex(workspaceID string) uint32 {
	// 空 workspace 也允许（历史/兜底写入），CRC32("")=0 → 落到 00 片，与迁移回填一致。
	return crc32.ChecksumIEEE([]byte(strings.TrimSpace(workspaceID))) % jobExecutionEventShardCount
}

func jobExecutionEventTable(workspaceID string) (string, error) {
	name := fmt.Sprintf("cloud_job_execution_event_%02d", jobExecutionEventShardIndex(workspaceID))
	if !jobExecutionEventTableNameRe.MatchString(name) {
		return "", fmt.Errorf("invalid job execution event shard table")
	}
	return name, nil
}

func jobExecutionEventShardTables() []string {
	out := make([]string, jobExecutionEventShardCount)
	for i := 0; i < jobExecutionEventShardCount; i++ {
		out[i] = fmt.Sprintf("cloud_job_execution_event_%02d", i)
	}
	return out
}
