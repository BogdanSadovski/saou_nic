package domain

import "time"

type CodeExecutionRequest struct {
	Language  string        `json:"language"`
	Code      string        `json:"code"`
	Input     string        `json:"input,omitempty"`
	Args      []string      `json:"args,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`
	TestCases []TestCase    `json:"test_cases,omitempty"`
}

type TestCase struct {
	Name     string `json:"name"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

type CodeExecutionResult struct {
	Status      ExecutionStatus `json:"status"`
	Output      string          `json:"output"`
	Error       string          `json:"error"`
	Runtime     time.Duration   `json:"runtime"`
	Memory      int64           `json:"memory"`
	TestResults []TestResult    `json:"test_results,omitempty"`
	ExitCode    int             `json:"exit_code"`
}

type ExecutionStatus string

const (
	StatusSuccess        ExecutionStatus = "success"
	StatusError          ExecutionStatus = "error"
	StatusTimeout        ExecutionStatus = "timeout"
	StatusMemoryExceeded ExecutionStatus = "memory_exceeded"
	StatusCompileError   ExecutionStatus = "compile_error"
	StatusRuntimeError   ExecutionStatus = "runtime_error"
)

type TestResult struct {
	Name     string        `json:"name"`
	Passed   bool          `json:"passed"`
	Expected string        `json:"expected"`
	Actual   string        `json:"actual"`
	Duration time.Duration `json:"duration"`
}

type Language string

const (
	LanguagePython     Language = "python"
	LanguageJavaScript Language = "javascript"
	LanguageGo         Language = "go"
	LanguageJava       Language = "java"
)

func (l Language) String() string {
	return string(l)
}

type ExecutionMetrics struct {
	ExecutionTime time.Duration
	MemoryUsed    int64
	OutputSize    int64
	CPUPercent    float64
}
