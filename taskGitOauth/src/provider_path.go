package main

import "strings"

func providerFromPath(path string) string {
	p := strings.ToLower(path)
	if strings.Contains(p, "gitlab") {
		return "gitlab"
	}
	return "github"
}

func normalizeProviderKey(bodyKey, pathProvider string) string {
	pk := strings.TrimSpace(strings.ToLower(bodyKey))
	if pk == "" {
		pk = pathProvider
	}
	if !strings.HasPrefix(pk, pathProvider+":") && pk != pathProvider {
		pk = pathProvider
	}
	return pk
}

// cacheKeyID 返回凭证缓存的键前缀：provider_key + remote_user_id。
// OPT-20260822-036: 同一 provider+user 下若绑定多个远端账号（不同 remote_user_id），
// 缓存/并发锁必须区分，否则换票会串账号。remote_user_id 为空时退化为 provider_key。
func cacheKeyID(providerKey, remoteUserID string) string {
	pk := strings.TrimSpace(strings.ToLower(providerKey))
	rid := strings.TrimSpace(remoteUserID)
	if rid == "" {
		return pk
	}
	return pk + "|r:" + rid
}
