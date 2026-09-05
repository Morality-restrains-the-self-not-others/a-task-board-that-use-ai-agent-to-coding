package repositories

import "context"

// CommitChangeRepository 定义提交变更查询的领域契约。
type CommitChangeRepository interface {
	ListHeadChangedFiles(ctx context.Context) ([]string, error)
}
