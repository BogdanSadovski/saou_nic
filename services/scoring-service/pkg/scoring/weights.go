package scoring

// WeightConfig defines the weight configuration for scoring calculations.
type WeightConfig struct {
	Criteria map[string]CriterionWeight `json:"criteria"`
}

// CriterionWeight holds the weight and bounds for a scoring criterion.
type CriterionWeight struct {
	Weight   float64 `json:"weight"`
	MinScore float64 `json:"min_score"`
	MaxScore float64 `json:"max_score"`
}

// DefaultWeights returns the default weight configuration for all score types.
func DefaultWeights() *WeightConfig {
	return &WeightConfig{
		Criteria: map[string]CriterionWeight{
			// Code quality weights
			"cyclomatic_complexity": {Weight: 0.30, MinScore: 0, MaxScore: 100},
			"code_duplication":      {Weight: 0.25, MinScore: 0, MaxScore: 100},
			"lint_compliance":       {Weight: 0.20, MinScore: 0, MaxScore: 100},
			"code_structure":        {Weight: 0.15, MinScore: 0, MaxScore: 100},
			"naming_conventions":    {Weight: 0.10, MinScore: 0, MaxScore: 100},
			// Performance weights
			"build_time":        {Weight: 0.30, MinScore: 0, MaxScore: 100},
			"memory_efficiency": {Weight: 0.35, MinScore: 0, MaxScore: 100},
			"response_time":     {Weight: 0.35, MinScore: 0, MaxScore: 100},
			// Security weights
			"vulnerability_count": {Weight: 0.50, MinScore: 0, MaxScore: 100},
			"input_validation":    {Weight: 0.30, MinScore: 0, MaxScore: 100},
			"auth_security":       {Weight: 0.20, MinScore: 0, MaxScore: 100},
			// Test coverage weights
			"line_coverage":   {Weight: 0.40, MinScore: 0, MaxScore: 100},
			"branch_coverage": {Weight: 0.35, MinScore: 0, MaxScore: 100},
			"test_quality":    {Weight: 0.25, MinScore: 0, MaxScore: 100},
			// Documentation weights
			"documentation_coverage": {Weight: 0.40, MinScore: 0, MaxScore: 100},
			"readme_quality":         {Weight: 0.30, MinScore: 0, MaxScore: 100},
			"api_documentation":      {Weight: 0.30, MinScore: 0, MaxScore: 100},
		},
	}
}

// CodeQualityWeights returns weights specific to code quality evaluation.
func CodeQualityWeights() map[string]float64 {
	return map[string]float64{
		"cyclomatic_complexity": 0.30,
		"code_duplication":      0.25,
		"lint_compliance":       0.20,
		"code_structure":        0.15,
		"naming_conventions":    0.10,
	}
}

// PerformanceWeights returns weights specific to performance evaluation.
func PerformanceWeights() map[string]float64 {
	return map[string]float64{
		"build_time":        0.30,
		"memory_efficiency": 0.35,
		"response_time":     0.35,
	}
}

// SecurityWeights returns weights specific to security evaluation.
func SecurityWeights() map[string]float64 {
	return map[string]float64{
		"vulnerability_count": 0.50,
		"input_validation":    0.30,
		"auth_security":       0.20,
	}
}

// TestCoverageWeights returns weights specific to test coverage evaluation.
func TestCoverageWeights() map[string]float64 {
	return map[string]float64{
		"line_coverage":   0.40,
		"branch_coverage": 0.35,
		"test_quality":    0.25,
	}
}

// DocumentationWeights returns weights specific to documentation evaluation.
func DocumentationWeights() map[string]float64 {
	return map[string]float64{
		"documentation_coverage": 0.40,
		"readme_quality":         0.30,
		"api_documentation":      0.30,
	}
}

// GetWeights returns the weight map for the given score type.
func GetWeights(scoreType string) map[string]float64 {
	switch scoreType {
	case "code_quality":
		return CodeQualityWeights()
	case "performance":
		return PerformanceWeights()
	case "security":
		return SecurityWeights()
	case "test_coverage":
		return TestCoverageWeights()
	case "documentation":
		return DocumentationWeights()
	default:
		return CodeQualityWeights()
	}
}

// NormalizeWeights ensures all weights sum to 1.0.
func NormalizeWeights(weights map[string]float64) map[string]float64 {
	total := 0.0
	for _, w := range weights {
		total += w
	}
	if total == 0 {
		return weights
	}

	normalized := make(map[string]float64)
	for k, w := range weights {
		normalized[k] = w / total
	}
	return normalized
}

// ApplyWeights calculates the weighted score from individual criterion scores.
func ApplyWeights(scores map[string]float64, weights map[string]float64) float64 {
	totalWeight := 0.0
	weightedSum := 0.0

	for criterion, score := range scores {
		w := 1.0
		if weight, ok := weights[criterion]; ok {
			w = weight
		}
		weightedSum += score * w
		totalWeight += w
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedSum / totalWeight
}

// CalculateGrade converts a numeric score to a letter grade.
func CalculateGrade(score float64) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 90:
		return "A"
	case score >= 85:
		return "A-"
	case score >= 80:
		return "B+"
	case score >= 75:
		return "B"
	case score >= 70:
		return "B-"
	case score >= 65:
		return "C+"
	case score >= 60:
		return "C"
	case score >= 55:
		return "C-"
	case score >= 50:
		return "D"
	default:
		return "F"
	}
}

// PassThreshold checks if the score meets the minimum passing threshold.
func PassThreshold(score, threshold float64) bool {
	return score >= threshold
}
