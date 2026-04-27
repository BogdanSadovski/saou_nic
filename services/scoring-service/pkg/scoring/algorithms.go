package scoring

import (
	"math"
	"strings"
)

// Algorithm defines a scoring algorithm interface for different evaluation strategies.
type Algorithm interface {
	Name() string
	Calculate(metrics Metrics) CriterionResults
}

// Metrics holds the raw metrics collected from a submission.
type Metrics struct {
	LOC                int     // Lines of code
	Complexity         float64 // Cyclomatic complexity average
	TestCoverage       float64 // Percentage of code covered by tests
	CodeDuplication    float64 // Percentage of duplicated code
	SecurityIssues     int     // Number of security vulnerabilities found
	DocumentationRatio float64 // Ratio of documentation to code
	LintErrors         int     // Number of linter errors
	BuildTime          float64 // Build time in seconds
	MemoryUsage        float64 // Peak memory usage in MB
	ResponseTime       float64 // Average response time in ms
}

// CriterionResult holds the result of evaluating a single criterion.
type CriterionResult struct {
	Name    string   `json:"name"`
	Score   float64  `json:"score"`
	MaxScore float64 `json:"max_score"`
	Comments []string `json:"comments,omitempty"`
}

// CodeQualityAlgorithm implements scoring for code quality metrics.
type CodeQualityAlgorithm struct{}

func (a *CodeQualityAlgorithm) Name() string {
	return "code_quality"
}

func (a *CodeQualityAlgorithm) Calculate(m Metrics) CriterionResults {
	results := CriterionResults{
		Results: make([]CriterionResult, 0, 5),
	}

	// Cyclomatic complexity scoring (lower is better)
	complexityScore := clamp(100-m.Complexity*5, 0, 100)
	results.Results = append(results.Results, CriterionResult{
		Name:     "cyclomatic_complexity",
		Score:    complexityScore,
		MaxScore: 100,
		Comments: complexityComments(m.Complexity),
	})

	// Code duplication scoring
	duplicationScore := clamp(100-m.CodeDuplication*2, 0, 100)
	results.Results = append(results.Results, CriterionResult{
		Name:     "code_duplication",
		Score:    duplicationScore,
		MaxScore: 100,
		Comments: duplicationComments(m.CodeDuplication),
	})

	// Lint compliance scoring
	lintScore := clamp(100-float64(m.LintErrors)*10, 0, 100)
	results.Results = append(results.Results, CriterionResult{
		Name:     "lint_compliance",
		Score:    lintScore,
		MaxScore: 100,
		Comments: lintComments(m.LintErrors),
	})

	// Code structure and organization
	structureScore := calculateStructureScore(m)
	results.Results = append(results.Results, CriterionResult{
		Name:     "code_structure",
		Score:    structureScore,
		MaxScore: 100,
		Comments: structureComments(structureScore),
	})

	// Naming conventions
	namingScore := calculateNamingScore(m)
	results.Results = append(results.Results, CriterionResult{
		Name:     "naming_conventions",
		Score:    namingScore,
		MaxScore: 100,
	})

	return results
}

// PerformanceAlgorithm implements scoring for performance metrics.
type PerformanceAlgorithm struct{}

func (a *PerformanceAlgorithm) Name() string {
	return "performance"
}

func (a *PerformanceAlgorithm) Calculate(m Metrics) CriterionResults {
	results := CriterionResults{
		Results: make([]CriterionResult, 0, 3),
	}

	// Build time scoring
	buildScore := clamp(100-m.BuildTime*2, 0, 100)
	results.Results = append(results.Results, CriterionResult{
		Name:     "build_time",
		Score:    buildScore,
		MaxScore: 100,
		Comments: buildTimeComments(m.BuildTime),
	})

	// Memory usage scoring
	memoryScore := clamp(100-m.MemoryUsage/10, 0, 100)
	results.Results = append(results.Results, CriterionResult{
		Name:     "memory_efficiency",
		Score:    memoryScore,
		MaxScore: 100,
		Comments: memoryComments(m.MemoryUsage),
	})

	// Response time scoring
	responseScore := clamp(100-m.ResponseTime/5, 0, 100)
	results.Results = append(results.Results, CriterionResult{
		Name:     "response_time",
		Score:    responseScore,
		MaxScore: 100,
		Comments: responseTimeComments(m.ResponseTime),
	})

	return results
}

// SecurityAlgorithm implements scoring for security analysis.
type SecurityAlgorithm struct{}

func (a *SecurityAlgorithm) Name() string {
	return "security"
}

func (a *SecurityAlgorithm) Calculate(m Metrics) CriterionResults {
	results := CriterionResults{
		Results: make([]CriterionResult, 0, 3),
	}

	// Vulnerability scoring (critical)
	vulnScore := clamp(100-float64(m.SecurityIssues)*25, 0, 100)
	results.Results = append(results.Results, CriterionResult{
		Name:     "vulnerability_count",
		Score:    vulnScore,
		MaxScore: 100,
		Comments: vulnerabilityComments(m.SecurityIssues),
	})

	// Input validation coverage
	inputValidationScore := calculateInputValidationScore(m)
	results.Results = append(results.Results, CriterionResult{
		Name:     "input_validation",
		Score:    inputValidationScore,
		MaxScore: 100,
	})

	// Authentication and authorization checks
	authScore := calculateAuthScore(m)
	results.Results = append(results.Results, CriterionResult{
		Name:     "auth_security",
		Score:    authScore,
		MaxScore: 100,
	})

	return results
}

// TestCoverageAlgorithm implements scoring for test coverage analysis.
type TestCoverageAlgorithm struct{}

func (a *TestCoverageAlgorithm) Name() string {
	return "test_coverage"
}

func (a *TestCoverageAlgorithm) Calculate(m Metrics) CriterionResults {
	results := CriterionResults{
		Results: make([]CriterionResult, 0, 3),
	}

	// Line coverage scoring
	lineCoverageScore := m.TestCoverage
	results.Results = append(results.Results, CriterionResult{
		Name:     "line_coverage",
		Score:    lineCoverageScore,
		MaxScore: 100,
		Comments: coverageComments(m.TestCoverage),
	})

	// Branch coverage (estimated from line coverage)
	branchCoverage := clamp(m.TestCoverage*0.8, 0, 100)
	results.Results = append(results.Results, CriterionResult{
		Name:     "branch_coverage",
		Score:    branchCoverage,
		MaxScore: 100,
	})

	// Test quality indicator (ratio of tests to LOC)
	testQuality := calculateTestQualityScore(m)
	results.Results = append(results.Results, CriterionResult{
		Name:     "test_quality",
		Score:    testQuality,
		MaxScore: 100,
	})

	return results
}

// DocumentationAlgorithm implements scoring for documentation quality.
type DocumentationAlgorithm struct{}

func (a *DocumentationAlgorithm) Name() string {
	return "documentation"
}

func (a *DocumentationAlgorithm) Calculate(m Metrics) CriterionResults {
	results := CriterionResults{
		Results: make([]CriterionResult, 0, 3),
	}

	// Documentation ratio scoring
	docScore := clamp(m.DocumentationRatio*200, 0, 100)
	results.Results = append(results.Results, CriterionResult{
		Name:     "documentation_coverage",
		Score:    docScore,
		MaxScore: 100,
		Comments: documentationComments(m.DocumentationRatio),
	})

	// README quality
	readmeScore := calculateReadmeScore(m)
	results.Results = append(results.Results, CriterionResult{
		Name:     "readme_quality",
		Score:    readmeScore,
		MaxScore: 100,
	})

	// API documentation
	apiDocScore := calculateApiDocScore(m)
	results.Results = append(results.Results, CriterionResult{
		Name:     "api_documentation",
		Score:    apiDocScore,
		MaxScore: 100,
	})

	return results
}

// CriterionResults holds a collection of criterion results.
type CriterionResults struct {
	Results []CriterionResult `json:"results"`
}

// TotalWeightedScore calculates the weighted total from all criterion results.
func (cr *CriterionResults) TotalWeightedScore(weights map[string]float64) float64 {
	if len(cr.Results) == 0 {
		return 0
	}

	totalWeight := 0.0
	weightedSum := 0.0

	for _, r := range cr.Results {
		w := 1.0
		if weight, ok := weights[r.Name]; ok {
			w = weight
		}
		weightedSum += (r.Score / r.MaxScore) * 100 * w
		totalWeight += w
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedSum / totalWeight
}

// Helper functions

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func complexityComments(complexity float64) []string {
	switch {
	case complexity < 5:
		return []string{"Excellent: Low complexity, easy to maintain"}
	case complexity < 10:
		return []string{"Good: Acceptable complexity level"}
	case complexity < 20:
		return []string{"Moderate: Consider refactoring complex functions"}
	default:
		return []string{"Poor: High complexity, refactoring strongly recommended"}
	}
}

func duplicationComments(duplication float64) []string {
	switch {
	case duplication < 3:
		return []string{"Excellent: Minimal code duplication"}
	case duplication < 10:
		return []string{"Good: Some duplication, consider extracting common code"}
	case duplication < 20:
		return []string{"Moderate: Significant duplication detected"}
	default:
		return []string{"Poor: Excessive duplication, apply DRY principle"}
	}
}

func lintComments(errors int) []string {
	switch {
	case errors == 0:
		return []string{"Excellent: No lint errors"}
	case errors < 5:
		return []string{"Good: Minor lint issues"}
	case errors < 15:
		return []string{"Moderate: Several lint violations"}
	default:
		return []string{"Poor: Many lint violations, address immediately"}
	}
}

func structureComments(score float64) []string {
	if score >= 80 {
		return []string{"Well-structured code with good organization"}
	}
	return []string{"Code structure could be improved"}
}

func calculateStructureScore(m Metrics) float64 {
	// Heuristic based on LOC and complexity ratio
	if m.LOC == 0 {
		return 50
	}
	ratio := m.Complexity / float64(m.LOC) * 1000
	return clamp(100-ratio*50, 0, 100)
}

func calculateNamingScore(m Metrics) float64 {
	// Base score; actual implementation would analyze AST
	baseScore := 75.0
	if m.LintErrors > 0 {
		baseScore -= float64(m.LintErrors) * 2
	}
	return clamp(baseScore, 0, 100)
}

func buildTimeComments(buildTime float64) []string {
	switch {
	case buildTime < 10:
		return []string{"Excellent: Fast build time"}
	case buildTime < 30:
		return []string{"Good: Acceptable build time"}
	case buildTime < 60:
		return []string{"Moderate: Build time could be optimized"}
	default:
		return []string{"Poor: Build time is too long"}
	}
}

func memoryComments(memory float64) []string {
	switch {
	case memory < 100:
		return []string{"Excellent: Low memory footprint"}
	case memory < 500:
		return []string{"Good: Reasonable memory usage"}
	case memory < 1000:
		return []string{"Moderate: Memory usage could be optimized"}
	default:
		return []string{"Poor: Excessive memory consumption"}
	}
}

func responseTimeComments(responseTime float64) []string {
	switch {
	case responseTime < 50:
		return []string{"Excellent: Very fast response"}
	case responseTime < 200:
		return []string{"Good: Acceptable response time"}
	case responseTime < 500:
		return []string{"Moderate: Response time could be improved"}
	default:
		return []string{"Poor: Slow response time"}
	}
}

func vulnerabilityComments(issues int) []string {
	switch {
	case issues == 0:
		return []string{"Excellent: No vulnerabilities found"}
	case issues <= 2:
		return []string{"Good: Few low-severity issues"}
	case issues <= 5:
		return []string{"Moderate: Several security concerns"}
	default:
		return []string{"Critical: Multiple security vulnerabilities detected"}
	}
}

func calculateInputValidationScore(m Metrics) float64 {
	// Heuristic: correlate with security issues
	if m.SecurityIssues == 0 {
		return 90
	}
	return clamp(100-float64(m.SecurityIssues)*15, 0, 100)
}

func calculateAuthScore(m Metrics) float64 {
	// Placeholder; real implementation would check auth patterns
	return 75.0
}

func coverageComments(coverage float64) []string {
	switch {
	case coverage >= 90:
		return []string{"Excellent: Comprehensive test coverage"}
	case coverage >= 70:
		return []string{"Good: Adequate test coverage"}
	case coverage >= 50:
		return []string{"Moderate: More tests needed"}
	default:
		return []string{"Poor: Insufficient test coverage"}
	}
}

func calculateTestQualityScore(m Metrics) float64 {
	if m.LOC == 0 {
		return 0
	}
	// Estimate test quality from coverage and LOC ratio
	testRatio := math.Sqrt(m.TestCoverage / 100)
	return clamp(testRatio*100, 0, 100)
}

func documentationComments(ratio float64) []string {
	switch {
	case ratio >= 0.3:
		return []string{"Excellent: Well documented"}
	case ratio >= 0.15:
		return []string{"Good: Adequate documentation"}
	case ratio >= 0.05:
		return []string{"Moderate: Documentation could be improved"}
	default:
		return []string{"Poor: Insufficient documentation"}
	}
}

func calculateReadmeScore(m Metrics) float64 {
	// Placeholder; real implementation would parse README
	return 70.0
}

func calculateApiDocScore(m Metrics) float64 {
	// Placeholder; real implementation would check API docs
	return 65.0
}

// GetAlgorithm returns the appropriate algorithm for a given score type.
func GetAlgorithm(scoreType string) Algorithm {
	switch strings.ToLower(scoreType) {
	case "code_quality":
		return &CodeQualityAlgorithm{}
	case "performance":
		return &PerformanceAlgorithm{}
	case "security":
		return &SecurityAlgorithm{}
	case "test_coverage":
		return &TestCoverageAlgorithm{}
	case "documentation":
		return &DocumentationAlgorithm{}
	default:
		return &CodeQualityAlgorithm{}
	}
}
