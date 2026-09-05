package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	AuthorSideTenant      = "tenant"
	AuthorSideSystemAdmin = "system_admin"
	maxOrderCommentRunes  = 2000
	defaultCommentLimit   = 100
)

var (
	ErrOrderCommentEmpty   = errors.New("评论内容不能为空")
	ErrOrderCommentTooLong = errors.New("评论内容不能超过 2000 字")
	ErrOrderCommentSide    = errors.New("无效的作者侧")
	ErrOrderCommentOrder   = errors.New("订单不存在或不属于该租户")
)

// OrderComment 订单评论实体
type OrderComment struct {
	ID           int64
	TenantID     int64
	OrderID      int64
	AuthorUserID string
	AuthorSide   string
	Content      string
	CreatedAt    string
}

func validateOrderCommentContent(content string) error {
	c := strings.TrimSpace(content)
	if c == "" {
		return ErrOrderCommentEmpty
	}
	if utf8.RuneCountInString(c) > maxOrderCommentRunes {
		return ErrOrderCommentTooLong
	}
	return nil
}

func validateAuthorSide(side string) error {
	switch side {
	case AuthorSideTenant, AuthorSideSystemAdmin:
		return nil
	default:
		return ErrOrderCommentSide
	}
}

// assertOrderBelongsToTenant 校验订单存在且归属租户。
func assertOrderBelongsToTenant(ctx context.Context, orderID, tenantID int64) error {
	var gotTenant int64
	err := db.QueryRowContext(ctx, `
		SELECT tenant_id FROM billing_resource_order WHERE id = ?`, orderID).Scan(&gotTenant)
	if err == sql.ErrNoRows {
		return ErrOrderCommentOrder
	}
	if err != nil {
		return err
	}
	if gotTenant != tenantID {
		return ErrOrderCommentOrder
	}
	return nil
}

func createOrderComment(ctx context.Context, tenantID, orderID int64, authorUserID, authorSide, content string) (*OrderComment, error) {
	if err := validateAuthorSide(authorSide); err != nil {
		return nil, err
	}
	if err := validateOrderCommentContent(content); err != nil {
		return nil, err
	}
	authorUserID = strings.TrimSpace(authorUserID)
	if authorUserID == "" {
		return nil, fmt.Errorf("缺少作者用户 ID")
	}
	if err := assertOrderBelongsToTenant(ctx, orderID, tenantID); err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	id := generateSnowflakeID()
	createdAt := utcNow()
	_, err := db.ExecContext(ctx, `
		INSERT INTO billing_order_comment
			(id, tenant_id, order_id, author_user_id, author_side, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, tenantID, orderID, authorUserID, authorSide, content, createdAt,
	)
	if err != nil {
		return nil, err
	}
	return &OrderComment{
		ID:           id,
		TenantID:     tenantID,
		OrderID:      orderID,
		AuthorUserID: authorUserID,
		AuthorSide:   authorSide,
		Content:      content,
		CreatedAt:    createdAt,
	}, nil
}

func listOrderComments(ctx context.Context, tenantID, orderID int64, limit int) ([]OrderComment, error) {
	if err := assertOrderBelongsToTenant(ctx, orderID, tenantID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > defaultCommentLimit {
		limit = defaultCommentLimit
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, order_id, author_user_id, author_side, content, created_at
		FROM billing_order_comment
		WHERE order_id = ? AND tenant_id = ?
		ORDER BY created_at ASC, id ASC
		LIMIT ?`, orderID, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]OrderComment, 0)
	for rows.Next() {
		var c OrderComment
		if err := rows.Scan(&c.ID, &c.TenantID, &c.OrderID, &c.AuthorUserID, &c.AuthorSide, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func orderCommentJSON(c *OrderComment) map[string]interface{} {
	return map[string]interface{}{
		"id":             formatID(c.ID),
		"tenant_id":      formatID(c.TenantID),
		"order_id":       formatID(c.OrderID),
		"author_user_id": c.AuthorUserID,
		"author_side":    c.AuthorSide,
		"content":        c.Content,
		"created_at":     c.CreatedAt,
	}
}
