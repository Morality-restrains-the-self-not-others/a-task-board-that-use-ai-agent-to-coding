package domain

// IdempotencyStorePort records successfully processed events.
type IdempotencyStorePort interface {
	Seen(key IdempotencyKey) bool
	Mark(key IdempotencyKey)
}
