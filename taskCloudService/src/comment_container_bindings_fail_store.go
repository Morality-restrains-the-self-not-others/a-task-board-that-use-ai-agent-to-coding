package main

import (
	"strings"
	"time"
)

// markCommentContainerBindingFailed 评论容器绑定永久失败（未配置云平台等）时收口 failed，
// 保留 mock 容器名与评论级 CSC 供 UI 定位（失败态不再参与 reachability / promote 闭环）。
// OPT-20260812-010：mock/空平台 bootstrap 永久失败时应收口 failed，避免无限停留在「启动中」。
func markCommentContainerBindingFailed(id, mockName, cscID string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	cscID = strings.TrimSpace(cscID)
	if cscID == "" {
		// 空 csc_id 不得覆盖已写入的评论级 CSC（启机失败路径常在 ensure CSC 之后才收口 failed）。
		_, err := db.Exec(
			`UPDATE cloud_comment_container_bindings
			SET status=?, mock_container_name=?, updated_at=?
			WHERE id=?`,
			ccbStatusFailed, mockName, now, id,
		)
		return err
	}
	_, err := db.Exec(
		`UPDATE cloud_comment_container_bindings
		SET status=?, mock_container_name=?, csc_id=?, updated_at=?
		WHERE id=?`,
		ccbStatusFailed, mockName, cscID, now, id,
	)
	return err
}
