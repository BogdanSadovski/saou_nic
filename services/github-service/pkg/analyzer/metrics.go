package analyzer

import (
	"math"
	"sort"
	"time"
)

// MetricType represents the type of metric being calculated
type MetricType string

const (
	MetricCommitFrequency    MetricType = "commit_frequency"
	MetricMergeRate          MetricType = "merge_rate"
	MetricBusFactor          MetricType = "bus_factor"
	MetricContributorGrowth  MetricType = "contributor_growth"
	MetricCodeChurn          MetricType = "code_churn"
	MetricReviewEfficiency   MetricType = "review_efficiency"
)

// MetricValue represents a calculated metric value
type MetricValue struct {
	Type      MetricType          `json:"type"`
	Value     float64             `json:"value"`
	Timestamp time.Time           `json:"timestamp"`
	Metadata  map[string]string   `json:"metadata,omitempty"`
}

// MetricsCalculator provides utilities for calculating various metrics
type MetricsCalculator struct {
	timeWindow time.Duration
}

// NewMetricsCalculator creates a new metrics calculator with the given time window
func NewMetricsCalculator(timeWindow time.Duration) *MetricsCalculator {
	return &MetricsCalculator{
		timeWindow: timeWindow,
	}
}

// CalculateTimeSeriesMetric calculates a metric over time series data
func (mc *MetricsCalculator) CalculateTimeSeriesMetric(values []float64, metricType MetricType) (*MetricValue, error) {
	if len(values) == 0 {
		return &MetricValue{
			Type:      metricType,
			Value:     0,
			Timestamp: time.Now(),
		}, nil
	}

	var result float64
	switch metricType {
	case MetricCommitFrequency:
		result = calculateMean(values)
	case MetricMergeRate:
		result = calculateMean(values)
	case MetricCodeChurn:
		result = calculateSum(values)
	default:
		result = calculateMean(values)
	}

	return &MetricValue{
		Type:      metricType,
		Value:     result,
		Timestamp: time.Now(),
	}, nil
}

// CalculatePercentile calculates the p-th percentile of a dataset
func (mc *MetricsCalculator) CalculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	index := (percentile / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// CalculateMovingAverage calculates a simple moving average
func (mc *MetricsCalculator) CalculateMovingAverage(values []float64, window int) []float64 {
	if len(values) == 0 || window <= 0 {
		return nil
	}

	if window > len(values) {
		window = len(values)
	}

	result := make([]float64, 0, len(values)-window+1)
	for i := 0; i <= len(values)-window; i++ {
		sum := 0.0
		for j := i; j < i+window; j++ {
			sum += values[j]
		}
		result = append(result, sum/float64(window))
	}

	return result
}

// CalculateTrend determines the trend direction of a time series
func (mc *MetricsCalculator) CalculateTrend(values []float64) TrendDirection {
	if len(values) < 2 {
		return TrendStable
	}

	// Simple linear regression slope
	n := float64(len(values))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, v := range values {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	switch {
	case slope > 0.1:
		return TrendIncreasing
	case slope < -0.1:
		return TrendDecreasing
	default:
		return TrendStable
	}
}

// TrendDirection represents the direction of a trend
type TrendDirection string

const (
	TrendIncreasing TrendDirection = "increasing"
	TrendDecreasing TrendDirection = "decreasing"
	TrendStable     TrendDirection = "stable"
)

// Normalize scales values to a 0-1 range
func Normalize(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}

	min := values[0]
	max := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if min == max {
		result := make([]float64, len(values))
		for i := range result {
			result[i] = 0.5
		}
		return result
	}

	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = (v - min) / (max - min)
	}
	return result
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateSum(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum
}
