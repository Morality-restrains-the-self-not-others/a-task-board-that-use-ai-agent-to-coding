package domain

// LoginMethodRepository defines persistence for login methods (implemented in main/sqlite).
type LoginMethodRepository interface {
	FindByEmail(email string) (*LoginMethodRef, error)
	FindByPhone(countryCode, national string) (*LoginMethodRef, error)
	FindByPasswordResetToken(token string) (*LoginMethodRef, error)
}
