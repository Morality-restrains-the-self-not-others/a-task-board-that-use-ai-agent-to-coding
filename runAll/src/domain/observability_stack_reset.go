package domain

import "context"

// ObservabilityStackResetResult is the aggregate outcome of a full observability reset.
type ObservabilityStackResetResult struct {
	Status                string
	MemoryServicesCleared int
	FilesTruncated        int
	LokiReset             string
	PromtailReset         string
	TempoReset            string
	PrometheusReset       string
}

// ObservabilityStackResetService orchestrates local log clearing and backend storage reset.
type ObservabilityStackResetService struct {
	logRepository ServiceLogRepository
	fileSink      ServiceLogFileSink
	storage       ObservabilityStorageResetter
}

func NewObservabilityStackResetService(
	logRepository ServiceLogRepository,
	fileSink ServiceLogFileSink,
	storage ObservabilityStorageResetter,
) *ObservabilityStackResetService {
	return &ObservabilityStackResetService{
		logRepository: logRepository,
		fileSink:      fileSink,
		storage:       storage,
	}
}

func (s *ObservabilityStackResetService) ClearAll(ctx context.Context, serviceNames []string) ObservabilityStackResetResult {
	result := ObservabilityStackResetResult{
		Status:        "ok",
		LokiReset:     "skipped",
		PromtailReset: "skipped",
		TempoReset:    "skipped",
		PrometheusReset: "skipped",
	}
	if s == nil {
		result.Status = "partial"
		return result
	}

	if s.logRepository != nil {
		for _, name := range serviceNames {
			if name == "" {
				continue
			}
			s.logRepository.Clear(name)
			result.MemoryServicesCleared++
		}
	}

	if s.fileSink != nil {
		if truncator, ok := s.fileSink.(ServiceLogFileTruncator); ok {
			count, err := truncator.TruncateAll()
			result.FilesTruncated = count
			if err != nil {
				result.Status = "partial"
			}
		}
	}

	if s.storage != nil {
		outcome, err := s.storage.Reset(ctx)
		result.LokiReset = outcome.LokiReset
		result.PromtailReset = outcome.PromtailReset
		result.TempoReset = outcome.TempoReset
		result.PrometheusReset = outcome.PrometheusReset
		if err != nil || !outcome.AllOK() {
			result.Status = "partial"
		}
	}

	return result
}
