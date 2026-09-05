package domain

import (
	"fmt"
	"time"
)

const (
	StatusDraft         = "draft"
	StatusPendingReview = "pending_review"
	StatusApproved      = "approved"
	StatusRejected      = "rejected"
)

var StatusDisplay = map[string]string{
	StatusDraft:         "草稿",
	StatusPendingReview: "待审核",
	StatusApproved:      "已上架",
	StatusRejected:      "已驳回",
}

type Vendor struct {
	ID           int64
	SaasUserID   *int64
	Email        string
	PasswordHash string
	CompanyName  string
	ContactName  string
	// 申请资质材料（vendor-application-kyc-docs）：本地 file_key + E.164 联系手机。
	IDCardFileKey          string
	BusinessLicenseFileKey string
	ContactPhone           string
	// 审核流字段（OPT-20260806-065）：is_active=0 且 ReviewNote 空 = 待审核(pending)；
	// ReviewNote 非空 = 已驳回(rejected)；is_active=1 = 已获准(qualified)。
	IsActive   bool
	ReviewNote string
	ReviewedAt *time.Time
	ReviewedBy *int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Staff struct {
	ID               int64
	SaasSuperadminID *int64
	Username         string
	PasswordHash     string
	DisplayName      string
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ImageGroup struct {
	ID          int64
	VendorID    int64
	Name        string
	Description string
	IconFileKey string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ContainerImage struct {
	ID                      int64
	VendorID                int64
	ImageGroupID            int64
	Version                 string
	SaasInboundSkillVersion string
	ImageURL                string
	TargetArchitecturesJSON string
	Size                    *int64
	Status                  string
	// IsActive 组内唯一激活标志（公开目录生效版本）。同一镜像组允许多个
	// approved，但仅一个激活版本；仅 approved 版本可被激活（OPT-20260824）。
	IsActive                  bool
	ReviewNote                string
	ReviewedAt                *time.Time
	ReviewerID                *int64
	AutoRunStepsMD            string
	AutoRunStepsExtractStatus string
	AutoRunStepsDigest        string
	ImageSkillsJSON           string
	ImageSkillsExtractStatus  string
	ImageSkillsDigest         string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

const (
	EventContainerImageReviewWithdrawn = "ContainerImageReviewWithdrawn"
	EventContainerImageUnpublished     = "ContainerImageUnpublished"
	EventContainerImageDeleted         = "ContainerImageDeleted"
)

func (c *ContainerImage) CanEdit() bool {
	return c.Status == StatusDraft || c.Status == StatusRejected
}

func (c *ContainerImage) CanDelete() bool {
	return c.DeleteGuard() == nil
}

// DeleteGuard 激活版本与审核中版本不可删。
func (c *ContainerImage) DeleteGuard() error {
	if c.IsActive {
		return fmt.Errorf("激活版本不可删除，请先下架或切换激活版本")
	}
	if c.Status == StatusPendingReview {
		return fmt.Errorf("审核中的版本请先撤回审核再删除")
	}
	if c.Status != StatusDraft && c.Status != StatusRejected && c.Status != StatusApproved {
		return fmt.Errorf("当前状态不可删除")
	}
	return nil
}

// VendorWithdraw 厂商撤回待审核或下架已上架；已是草稿则空操作。
func (c *ContainerImage) VendorWithdraw(actorID int64, note string, now time.Time) error {
	switch c.Status {
	case StatusPendingReview:
		return c.WithdrawPending()
	case StatusApproved:
		if note == "" {
			note = "厂商自行下架"
		}
		return c.Unpublish(actorID, note, now)
	case StatusDraft:
		return nil
	default:
		return fmt.Errorf("当前状态不可撤回或下架")
	}
}

func (c *ContainerImage) Submit() error {
	if c.Status != StatusDraft && c.Status != StatusRejected {
		return fmt.Errorf("当前状态不可提交审核")
	}
	c.Status = StatusPendingReview
	c.ReviewNote = ""
	return nil
}

func (c *ContainerImage) Approve(reviewerID int64, note string, now time.Time) error {
	if c.Status != StatusPendingReview {
		return fmt.Errorf("仅待审核镜像可通过审批")
	}
	c.Status = StatusApproved
	c.ReviewerID = &reviewerID
	c.ReviewedAt = &now
	c.ReviewNote = note
	return nil
}

func (c *ContainerImage) Reject(reviewerID int64, note string, now time.Time) error {
	if c.Status != StatusPendingReview {
		return fmt.Errorf("仅待审核镜像可驳回")
	}
	if note == "" {
		return fmt.Errorf("请填写驳回原因")
	}
	c.Status = StatusRejected
	c.ReviewerID = &reviewerID
	c.ReviewedAt = &now
	c.ReviewNote = note
	return nil
}

// WithdrawPending lets vendor cancel a pending review (pending_review → draft).
func (c *ContainerImage) WithdrawPending() error {
	if c.Status != StatusPendingReview {
		return fmt.Errorf("仅待审核可撤回")
	}
	c.Status = StatusDraft
	return nil
}

// Unpublish lets staff take an approved image off the catalog (approved → draft).
func (c *ContainerImage) Unpublish(reviewerID int64, note string, now time.Time) error {
	if c.Status != StatusApproved {
		return fmt.Errorf("仅已上架镜像可下架")
	}
	c.Status = StatusDraft
	c.IsActive = false
	c.ReviewerID = &reviewerID
	c.ReviewedAt = &now
	c.ReviewNote = note
	return nil
}

// CanActivate 仅已上架（审批通过）的版本可被设为组内激活版本。
func (c *ContainerImage) CanActivate() bool {
	return c.Status == StatusApproved
}

// Withdraw is staff revoke of approved listing (alias of Unpublish for API naming).
func (c *ContainerImage) Withdraw(reviewerID int64, note string, now time.Time) error {
	return c.Unpublish(reviewerID, note, now)
}

type CloudServerImage struct {
	ID                       int64
	VendorID                 int64
	PlatformType             string
	ImageName                string
	ImageID                  string
	Region                   string
	OSType                   string
	OSVersion                string
	Architecture             string
	ImageType                string
	ImageSizeGB              *int
	IsActive                 bool
	DefaultInstanceTypeID    string
	DefaultInstanceTypeLabel string
	BaseCPUCores             *int
	BaseMemoryGiB            *int
	UserDataTemplateID       *int64
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type UserDataTemplate struct {
	ID                int64
	Name              string
	Version           string
	OSType            string
	VariablesJSON     string
	ContainerVarsJSON string
	Content           string
	AutoVerifyScript  string
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Association struct {
	ID                 int64
	ContainerImageID   int64
	CloudServerImageID int64
	PlatformType       string
	Region             string
}

type ReviewHistory struct {
	ID               int64
	ContainerImageID int64
	Action           string
	Note             string
	ReviewerID       *int64
	ReviewedAt       time.Time
}
