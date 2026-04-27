package service

import (
	"context"
	"fmt"

	"scoring-service/internal/domain"
	"scoring-service/pkg/scoring"
)

// EvaluationEngine orchestrates the metric collection and algorithm execution.
type EvaluationEngine struct {
	algorithms map[string]scoring.Algorithm
}

// NewEvaluationEngine creates a new evaluation engine with registered algorithms.
func NewEvaluationEngine() *EvaluationEngine {
	return &EvaluationEngine{
		algorithms: map[string]scoring.Algorithm{
			"code_quality":    &scoring.CodeQualityAlgorithm{},
			"performance":     &scoring.PerformanceAlgorithm{},
			"security":        &scoring.SecurityAlgorithm{},
			"test_coverage":   &scoring.TestCoverageAlgorithm{},
			"documentation":   &scoring.DocumentationAlgorithm{},
		},
	}
}

// CollectMetrics gathers metrics from the submission for evaluation.
// In a real implementation, this would integrate with CI/CD tools,
// static analyzers, and other measurement systems.
func (e *EvaluationEngine) CollectMetrics(ctx context.Context, req domain.ScoringRequest) (*scoring.Metrics, error) {
	// TODO: Integrate with actual metric collection systems
	// This is a placeholder that would be replaced with real metric collection
	// from tools like:
	// - SonarQube / SonarCloud for code quality
	// - Go test -cover for test coverage
	// - Gosec for security analysis
	// - Go benchmark for performance metrics

	metrics := &scoring.Metrics{
		LOC:                1000,
		Complexity:         8.5,
		TestCoverage:       72.5,
		CodeDuplication:    5.2,
		SecurityIssues:     1,
		DocumentationRatio: 0.12,
		LintErrors:         3,
		BuildTime:          15.3,
		MemoryUsage:        256.0,
		ResponseTime:       120.0,
	}

	return metrics, nil
}

// Evaluate runs the appropriate algorithm for the given score type.
func (e *EvaluationEngine) Evaluate(ctx context.Context, scoreType domain.ScoreType, metrics scoring.Metrics) (scoring.CriterionResults, error) {
	algorithm, ok := e.algorithms[string(scoreType)]
	if !ok {
		return scoring.CriterionResults{}, fmt.Errorf("no algorithm registered for score type: %s", scoreType)
	}

	results := algorithm.Calculate(metrics)
	return results, nil
}

// EvaluateAll runs all registered algorithms and returns aggregated results.
func (e *EvaluationEngine) EvaluateAll(ctx context.Context, metrics scoring.Metrics) map[string]scoring.CriterionResults {
	results := make(map[string]scoring.CriterionResults)

	for scoreType, algorithm := range e.algorithms {
		results[scoreType] = algorithm.Calculate(metrics)
	}

	return results
}

// RegisterAlgorithm adds a new algorithm to the evaluation engine.
func (e *EvaluationEngine) RegisterAlgorithm(scoreType string, algorithm scoring.Algorithm) {
	e.algorithms[scoreType] = algorithm
}

// ListAlgorithms returns the names of all registered algorithms.
func (e *EvaluationEngine) ListAlgorithms() []string {
	types := make([]string, 0, len(e.algorithms))
	for t := range e.algorithms {
		types = append(types, t)
	}
	return types
}
