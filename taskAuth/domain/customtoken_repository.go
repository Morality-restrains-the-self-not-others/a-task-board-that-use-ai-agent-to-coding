package domain

// CustomTokenRepository 定义认证令牌的持久化契约。
// 实现在 main/sqlite。
type CustomTokenRepository interface {
	// DeleteByKey 按令牌密钥删除记录。
	// 令牌不存在时不报错（幂等）。
	DeleteByKey(key string) error
}
