package domain

import (
	"fmt"
	"net/url"
)

// SessionTermination 表示一次会话终止操作。
// 它封装了 OIDC EndSession 后的重定向链：
//
//	taskAuth → GitLab sign_out → post_logout_redirect_uri
//
// 根据 NFR 要求（L1 一致性，best-effort），GitLab 不可达时仍返回 sign_out URL，
// 不做可达性检查。
type SessionTermination struct {
	// PostLogoutRedirectURI 是最终回跳地址（如主应用 /auth/login/）
	PostLogoutRedirectURI string
	// GitServicePublicBase 是 GitLab 的外部 URL（如 http://<gitlab-host>:8012）
	GitServicePublicBase string
}

// NewSessionTermination 创建会话终止值对象。
// gitServicePublicBase 不可为空。
func NewSessionTermination(gitServicePublicBase, postLogoutRedirectURI string) (*SessionTermination, error) {
	if gitServicePublicBase == "" {
		return nil, fmt.Errorf("domain: gitServicePublicBase must not be empty")
	}
	return &SessionTermination{
		PostLogoutRedirectURI: postLogoutRedirectURI,
		GitServicePublicBase:  gitServicePublicBase,
	}, nil
}

// GitLabSignOutURL 构造 GitLab 登出 URL。
// 如果设置了 PostLogoutRedirectURI，GitLab 登出后会回跳至该地址。
// 格式: {gitServicePublicBase}/users/sign_out?redirect_uri={postLogoutRedirectURI}
func (st *SessionTermination) GitLabSignOutURL() string {
	signOutURL := st.GitServicePublicBase + "/users/sign_out"
	if st.PostLogoutRedirectURI != "" {
		signOutURL += "?redirect_uri=" + url.QueryEscape(st.PostLogoutRedirectURI)
	}
	return signOutURL
}

// Equals 值相等比较。
func (st *SessionTermination) Equals(other *SessionTermination) bool {
	if other == nil {
		return false
	}
	return st.GitServicePublicBase == other.GitServicePublicBase &&
		st.PostLogoutRedirectURI == other.PostLogoutRedirectURI
}
