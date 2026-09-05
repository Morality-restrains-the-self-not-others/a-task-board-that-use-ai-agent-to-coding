package domain

// ServiceLogFileTruncator truncates persisted tee log files for observability reset.
type ServiceLogFileTruncator interface {
	TruncateService(serviceName string) error
	TruncateAll() (int, error)
}
