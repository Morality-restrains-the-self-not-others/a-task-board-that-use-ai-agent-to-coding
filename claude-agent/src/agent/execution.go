// Package agent defines the Agent execution types for claude-agent.
package agent

import "encoding/json"

// State represents the overall agent execution state.
type State string

const (
	StateIdle      State = "idle"
	StateRunning   State = "running"
	StateCompleted State = "completed"
	StateError     State = "error"
)

// StepState represents the state of an individual agent step.
type StepState string

const (
	StepThinking    StepState = "thinking"
	StepCallingTool StepState = "calling_tool"
	StepReflecting  StepState = "reflecting"
	StepCompleted   StepState = "completed"
	StepError       StepState = "error"
)

// Execution tracks a full agent task execution.
type Execution struct {
	Task          string  `json:"task"`
	Steps         []Step  `json:"steps"`
	State         State   `json:"agent_state"`
	Success       bool    `json:"success"`
	FinalResult   string  `json:"final_result,omitempty"`
	ExecutionTime float64 `json:"execution_time"`
	ErrorMessage  string  `json:"error_message,omitempty"`
	TotalTokens   *Tokens `json:"total_tokens,omitempty"`
}

// Step represents a single step in agent execution.
type Step struct {
	StepNumber  int       `json:"step_number"`
	State       StepState `json:"state"`
	LLMResponse string    `json:"llm_response,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// Tokens tracks token usage.
type Tokens struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ToJSON serializes the execution to JSON bytes.
func (e *Execution) ToJSON() ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}
