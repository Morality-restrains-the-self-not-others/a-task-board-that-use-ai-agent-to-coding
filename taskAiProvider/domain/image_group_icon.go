package domain

import "strings"

func RequireImageGroupFields(name, description, iconFileKey string) error {
	if strings.TrimSpace(name) == "" {
		return fmtImageGroupRequired("镜像组名称必填")
	}
	if strings.TrimSpace(description) == "" {
		return fmtImageGroupRequired("镜像组描述必填")
	}
	if strings.TrimSpace(iconFileKey) == "" {
		return fmtImageGroupRequired("镜像组图标必填")
	}
	return nil
}

type imageGroupRequiredError string

func (e imageGroupRequiredError) Error() string { return string(e) }

func fmtImageGroupRequired(msg string) error { return imageGroupRequiredError(msg) }
