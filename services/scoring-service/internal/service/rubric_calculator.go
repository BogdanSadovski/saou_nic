package service

import (
	"scoring-service/internal/domain"
	"scoring-service/pkg/scoring"
)

// RubricCalculator handles rubric-based score calculations.
type RubricCalculator struct{}

// NewRubricCalculator creates a new rubric calculator.
func NewRubricCalculator() *RubricCalculator {
	return &RubricCalculator{}
}

// CalculateFromRubric evaluates a submission against a rubric's criteria.
func (c *RubricCalculator) CalculateFromRubric(rubric *domain.Rubric, metrics scoring.Metrics) []domain.CriterionScore {
	scores := make([]domain.CriterionScore, 0, len(rubric.Criteria))

	for _, criterion := range rubric.Criteria {
		// Evaluate each criterion based on its name and the available metrics
		rawScore := c.evaluateCriterion(criterion.Name, metrics)

		// Normalize the score to the criterion's max score
		normalizedScore := (rawScore / 100) * criterion.MaxScore
		weightedScore := normalizedScore * criterion.Weight

		scores = append(scores, domain.CriterionScore{
			CriterionName: criterion.Name,
			Score:         normalizedScore,
			MaxScore:      criterion.MaxScore,
			Weight:        criterion.Weight,
			WeightedScore: weightedScore,
			Comments:      c.generateCriterionComments(criterion.Name, rawScore),
		})
	}

	return scores
}

// EvaluateCriterionWithWeights calculates scores using custom weights.
func (c *RubricCalculator) EvaluateCriterionWithWeights(criteria []domain.RubricCriterion, metrics scoring.Metrics) map[string]float64 {
	scores := make(map[string]float64)

	for _, criterion := range criteria {
		rawScore := c.evaluateCriterion(criterion.Name, metrics)
		scores[criterion.Name] = rawScore
	}

	return scores
}

// CalculateWeightedTotal computes the final weighted score from criterion scores.
func (c *RubricCalculator) CalculateWeightedTotal(scores []domain.CriterionScore) float64 {
	totalWeight := 0.0
	weightedSum := 0.0

	for _, cs := range scores {
		w := cs.Weight
		if w == 0 {
			w = 1.0
		}
		weightedSum += (cs.Score / cs.MaxScore) * 100 * w
		totalWeight += w
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedSum / totalWeight
}

// evaluateCriterion maps a criterion name to the appropriate metric calculation.
func (c *RubricCalculator) evaluateCriterion(name string, metrics scoring.Metrics) float64 {
	switch name {
	case "complexity", "cyclomatic_complexity":
		return clamp(100-metrics.Complexity*5, 0, 100)
	case "duplication", "code_duplication":
		return clamp(100-metrics.CodeDuplication*2, 0, 100)
	case "test_coverage", "coverage":
		return metrics.TestCoverage
	case "documentation", "doc_coverage":
		return clamp(metrics.DocumentationRatio*200, 0, 100)
	case "security", "vulnerabilities":
		return clamp(100-float64(metrics.SecurityIssues)*25, 0, 100)
	case "lint", "lint_compliance":
		return clamp(100-float64(metrics.LintErrors)*10, 0, 100)
	case "performance", "response_time":
		return clamp(100-metrics.ResponseTime/5, 0, 100)
	case "memory", "memory_usage":
		return clamp(100-metrics.MemoryUsage/10, 0, 100)
	case "build_time":
		return clamp(100-metrics.BuildTime*2, 0, 100)
	case "code_structure":
		if metrics.LOC == 0 {
			return 50
		}
		ratio := metrics.Complexity / float64(metrics.LOC) * 1000
		return clamp(100-ratio*50, 0, 100)
	default:
		// Default score for unknown criteria
		return 50.0
	}
}

// generateCriterionComments provides feedback based on criterion score.
func (c *RubricCalculator) generateCriterionComments(name string, score float64) []string {
	switch {
	case score >= 90:
		return []string{name + ": Excellent performance"}
	case score >= 75:
		return []string{name + ": Good, minor improvements possible"}
	case score >= 60:
		return []string{name + ": Acceptable, room for improvement"}
	case score >= 40:
		return []string{name + ": Below average, needs attention"}
	default:
		return []string{name + ": Poor, requires immediate improvement"}
	}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
