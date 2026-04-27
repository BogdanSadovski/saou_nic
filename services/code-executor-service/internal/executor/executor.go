package executor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"code-executor-service/internal/config"
	"code-executor-service/internal/domain"
)

type Executor struct {
	cfg    *config.Config
	logger *log.Logger
}

func New(cfg *config.Config) *Executor {
	return &Executor{
		cfg:    cfg,
		logger: log.New(os.Stdout, "code-executor: ", log.LstdFlags),
	}
}

func (e *Executor) Execute(ctx context.Context, req *domain.CodeExecutionRequest) *domain.CodeExecutionResult {
	// Validate request
	if err := e.validateRequest(req); err != nil {
		return &domain.CodeExecutionResult{
			Status:   domain.StatusError,
			Error:    err.Error(),
			ExitCode: 1,
		}
	}

	// Set timeout
	timeout := e.cfg.Executor.MaxExecutionTime
	if req.Timeout > 0 && req.Timeout < timeout {
		timeout = req.Timeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	switch domain.Language(req.Language) {
	case domain.LanguagePython:
		return e.executePython(ctx, req, start)
	case domain.LanguageJavaScript:
		return e.executeJavaScript(ctx, req, start)
	case domain.LanguageGo:
		return e.executeGo(ctx, req, start)
	case domain.LanguageJava:
		return e.executeJava(ctx, req, start)
	default:
		return &domain.CodeExecutionResult{
			Status:   domain.StatusError,
			Error:    fmt.Sprintf("unsupported language: %s", req.Language),
			ExitCode: 1,
		}
	}
}

func (e *Executor) validateRequest(req *domain.CodeExecutionRequest) error {
	if req.Language == "" {
		return errors.New("language is required")
	}

	if req.Code == "" {
		return errors.New("code is required")
	}

	if len(req.Code) > int(e.cfg.Security.MaxCodeSize) {
		return fmt.Errorf("code size exceeds limit: %d > %d", len(req.Code), e.cfg.Security.MaxCodeSize)
	}

	// Check for disallowed patterns
	for _, pattern := range e.cfg.Security.DisallowedPatterns {
		if strings.Contains(req.Code, pattern) {
			return fmt.Errorf("disallowed pattern detected: %s", pattern)
		}
	}

	return nil
}

func (e *Executor) executePython(ctx context.Context, req *domain.CodeExecutionRequest, start time.Time) *domain.CodeExecutionResult {
	// Create temp file for Python code
	tmpFile, err := os.CreateTemp("", "python_*.py")
	if err != nil {
		return &domain.CodeExecutionResult{
			Status:   domain.StatusError,
			Error:    fmt.Sprintf("failed to create temp file: %v", err),
			ExitCode: 1,
		}
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(req.Code); err != nil {
		return &domain.CodeExecutionResult{
			Status:   domain.StatusError,
			Error:    fmt.Sprintf("failed to write code: %v", err),
			ExitCode: 1,
		}
	}
	tmpFile.Close()

	cmd := exec.CommandContext(ctx, e.cfg.Executor.Python.Binary, tmpFile.Name())
	cmd.Stdin = strings.NewReader(req.Input)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &domain.CodeExecutionResult{
				Status:   domain.StatusTimeout,
				Error:    "execution timeout exceeded",
				Runtime:  time.Since(start),
				Output:   string(output),
				ExitCode: 124,
			}
		}

		result := &domain.CodeExecutionResult{
			Status:  domain.StatusRuntimeError,
			Error:   err.Error(),
			Output:  string(output),
			Runtime: time.Since(start),
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result
	}

	return &domain.CodeExecutionResult{
		Status:   domain.StatusSuccess,
		Output:   string(output),
		Runtime:  time.Since(start),
		ExitCode: 0,
	}
}

func (e *Executor) executeJavaScript(ctx context.Context, req *domain.CodeExecutionRequest, start time.Time) *domain.CodeExecutionResult {
	cmd := exec.CommandContext(ctx, e.cfg.Executor.JavaScript.Binary, "-e", req.Code)
	cmd.Stdin = strings.NewReader(req.Input)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &domain.CodeExecutionResult{
				Status:   domain.StatusTimeout,
				Error:    "execution timeout exceeded",
				Runtime:  time.Since(start),
				Output:   string(output),
				ExitCode: 124,
			}
		}

		result := &domain.CodeExecutionResult{
			Status:  domain.StatusRuntimeError,
			Error:   err.Error(),
			Output:  string(output),
			Runtime: time.Since(start),
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result
	}

	return &domain.CodeExecutionResult{
		Status:   domain.StatusSuccess,
		Output:   string(output),
		Runtime:  time.Since(start),
		ExitCode: 0,
	}
}

func (e *Executor) executeGo(ctx context.Context, req *domain.CodeExecutionRequest, start time.Time) *domain.CodeExecutionResult {
	// Go execution requires compiltion first - for now, return not implemented
	return &domain.CodeExecutionResult{
		Status:   domain.StatusError,
		Error:    "Go compilation not yet implemented in this version",
		ExitCode: 1,
	}
}

func (e *Executor) executeJava(ctx context.Context, req *domain.CodeExecutionRequest, start time.Time) *domain.CodeExecutionResult {
	// Java execution requires compiltion first - for now, return not implemented
	return &domain.CodeExecutionResult{
		Status:   domain.StatusError,
		Error:    "Java compilation not yet implemented in this version",
		ExitCode: 1,
	}
}

func (e *Executor) CompileAndExecute(ctx context.Context, req *domain.CodeExecutionRequest) *domain.CodeExecutionResult {
	// For compiled languages (Go, Java), compile first then execute
	// This will be implemented in Phase 2
	return &domain.CodeExecutionResult{
		Status:   domain.StatusError,
		Error:    "compiled language support coming in Phase 2",
		ExitCode: 1,
	}
}
