package main

import "strings"

// canonicalGitRepoURL 用于比较任务记录仓库地址与项目当前 git_repos。
// 忽略首尾空白、尾斜杠、.git 后缀，并小写化（GitHub owner/repo 大小写不敏感）。
func canonicalGitRepoURL(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimRight(s, "/")
	if len(s) >= 4 && strings.EqualFold(s[len(s)-4:], ".git") {
		s = s[:len(s)-4]
	}
	return strings.ToLower(s)
}

func gitRepoURLsEqual(a, b string) bool {
	ca, cb := canonicalGitRepoURL(a), canonicalGitRepoURL(b)
	return ca != "" && ca == cb
}

func indexOfGitRepoURL(urls []string, addr string) int {
	for i, u := range urls {
		if gitRepoURLsEqual(u, addr) {
			return i
		}
	}
	return -1
}
