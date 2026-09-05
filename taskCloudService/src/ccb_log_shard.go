package main

import (
	"fmt"
	"hash/crc32"
	"regexp"
	"strings"
)

const ccbLogShardCount = 16

var ccbLogTableNameRe = regexp.MustCompile(`^cloud_comment_container_binding_logs_[0-9]{2}$`)

func ccbLogShardIndex(workspaceID string) (uint32, error) {
	ws := strings.TrimSpace(workspaceID)
	if ws == "" {
		return 0, fmt.Errorf("workspace_id required for log shard")
	}
	return crc32.ChecksumIEEE([]byte(ws)) % ccbLogShardCount, nil
}

func ccbLogTable(workspaceID string) (string, error) {
	idx, err := ccbLogShardIndex(workspaceID)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("cloud_comment_container_binding_logs_%02d", idx)
	if !ccbLogTableNameRe.MatchString(name) {
		return "", fmt.Errorf("invalid ccb log shard table")
	}
	return name, nil
}

func ccbLogShardTables() []string {
	out := make([]string, ccbLogShardCount)
	for i := 0; i < ccbLogShardCount; i++ {
		out[i] = fmt.Sprintf("cloud_comment_container_binding_logs_%02d", i)
	}
	return out
}
