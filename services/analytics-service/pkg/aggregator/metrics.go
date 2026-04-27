package aggregator

import (
	"fmt"
	"math"
	"sort"
	"time"

	"analytics-service/internal/domain"
)

// MetricsAggregator computes aggregated metrics from raw events.
type MetricsAggregator struct{}

// NewMetricsAggregator creates a new MetricsAggregator.
func NewMetricsAggregator() *MetricsAggregator {
	return &MetricsAggregator{}
}

// Aggregate computes aggregated metrics for a set of events within a time window.
func (a *MetricsAggregator) Aggregate(
	events []*domain.Event,
	windowStart, windowEnd time.Time,
	granularity string,
) []*domain.AggregatedMetrics {
	if len(events) == 0 {
		return nil
	}

	// Group events by tenant.
	byTenant := make(map[string][]*domain.Event)
	for _, e := range events {
		byTenant[e.TenantID] = append(byTenant[e.TenantID], e)
	}

	var result []*domain.AggregatedMetrics
	for tenantID, tenantEvents := range byTenant {
		metrics := a.computeTenantMetrics(tenantID, tenantEvents, windowStart, windowEnd, granularity)
		result = append(result, metrics)
	}

	return result
}

// AggregateByInterval computes metrics for each time interval within the window.
func (a *MetricsAggregator) AggregateByInterval(
	events []*domain.Event,
	windowStart, windowEnd time.Time,
	granularity string,
) []*domain.AggregatedMetrics {
	if len(events) == 0 {
		return nil
	}

	// Determine interval duration.
	intervalDuration := a.granularityToDuration(granularity)

	// Group events by tenant and time bucket.
	type bucketKey struct {
		TenantID string
		Bucket   time.Time
	}
	buckets := make(map[bucketKey][]*domain.Event)

	for _, e := range events {
		bucket := a.bucketTime(e.Timestamp, intervalDuration)
		if bucket.Before(windowStart) || bucket.After(windowEnd) {
			continue
		}

		key := bucketKey{TenantID: e.TenantID, Bucket: bucket}
		buckets[key] = append(buckets[key], e)
	}

	var result []*domain.AggregatedMetrics
	for key, bucketEvents := range buckets {
		windowEndBucket := key.Bucket.Add(intervalDuration)
		metrics := a.computeTenantMetrics(key.TenantID, bucketEvents, key.Bucket, windowEndBucket, granularity)
		result = append(result, metrics)
	}

	// Sort by window start.
	sort.Slice(result, func(i, j int) bool {
		return result[i].WindowStart.Before(result[j].WindowStart)
	})

	return result
}

func (a *MetricsAggregator) computeTenantMetrics(
	tenantID string,
	events []*domain.Event,
	windowStart, windowEnd time.Time,
	granularity string,
) *domain.AggregatedMetrics {
	metrics := &domain.AggregatedMetrics{
		ID:          generateMetricsID(tenantID, windowStart, granularity),
		TenantID:    tenantID,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		Granularity: granularity,
		CreatedAt:   time.Now().UTC(),
	}

	uniqueUsers := make(map[string]struct{})
	uniqueSessions := make(map[string]struct{})
	var sessionDurations []float64
	var errors int64

	for _, e := range events {
		metrics.TotalEvents++

		if e.UserID != "" {
			uniqueUsers[e.UserID] = struct{}{}
		}
		if e.SessionID != "" {
			uniqueSessions[e.SessionID] = struct{}{}
		}

		switch e.Type {
		case domain.EventPageView:
			metrics.PageViews++
		case domain.EventClick:
			metrics.Clicks++
		case domain.EventPurchase, domain.EventSignup:
			metrics.Conversions++
		case domain.EventError:
			errors++
		}

		// Extract session duration if available.
		if dur, ok := e.Properties["session_duration"]; ok {
			if d, ok := dur.(float64); ok {
				sessionDurations = append(sessionDurations, d)
			}
		}
	}

	metrics.UniqueUsers = int64(len(uniqueUsers))
	metrics.UniqueSessions = int64(len(uniqueSessions))
	metrics.Errors = errors

	// Compute average session duration.
	if len(sessionDurations) > 0 {
		var sum float64
		for _, d := range sessionDurations {
			sum += d
		}
		metrics.AvgSessionDuration = sum / float64(len(sessionDurations))
	}

	// Compute bounce rate (sessions with only 1 event).
	if metrics.UniqueSessions > 0 {
		sessionEventCount := make(map[string]int)
		for _, e := range events {
			if e.SessionID != "" {
				sessionEventCount[e.SessionID]++
			}
		}

		var bounces int64
		for _, count := range sessionEventCount {
			if count == 1 {
				bounces++
			}
		}

		metrics.BounceRate = float64(bounces) / float64(metrics.UniqueSessions) * 100
	}

	return metrics
}

// ComputeTimeSeries computes time series data points from events.
func (a *MetricsAggregator) ComputeTimeSeries(
	events []*domain.Event,
	granularity string,
) []domain.TimeSeriesPoint {
	if len(events) == 0 {
		return nil
	}

	intervalDuration := a.granularityToDuration(granularity)

	// Bucket events by time.
	buckets := make(map[time.Time]int64)
	for _, e := range events {
		bucket := a.bucketTime(e.Timestamp, intervalDuration)
		buckets[bucket]++
	}

	// Sort by time.
	timestamps := make([]time.Time, 0, len(buckets))
	for t := range buckets {
		timestamps = append(timestamps, t)
	}
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

	// Build time series.
	points := make([]domain.TimeSeriesPoint, 0, len(timestamps))
	for _, t := range timestamps {
		points = append(points, domain.TimeSeriesPoint{
			Timestamp: t,
			Value:     float64(buckets[t]),
		})
	}

	return points
}

// ComputeEventDistribution computes the distribution of event types.
func (a *MetricsAggregator) ComputeEventDistribution(events []*domain.Event) map[string]float64 {
	if len(events) == 0 {
		return nil
	}

	typeCounts := make(map[domain.EventType]int64)
	for _, e := range events {
		typeCounts[e.Type]++
	}

	total := float64(len(events))
	distribution := make(map[string]float64)
	for eventType, count := range typeCounts {
		distribution[string(eventType)] = float64(count) / total * 100
	}

	return distribution
}

// ComputeTopItems computes the most frequent items for a given property.
func (a *MetricsAggregator) ComputeTopItems(events []*domain.Event, property string, limit int) []map[string]interface{} {
	if len(events) == 0 {
		return nil
	}

	counts := make(map[string]int64)
	for _, e := range events {
		var value string
		switch property {
		case "url":
			value = e.URL
		case "country":
			value = e.Country
		case "device":
			value = e.Device
		case "os":
			value = e.OS
		case "browser":
			value = e.Browser
		case "referrer":
			value = e.Referrer
		default:
			if v, ok := e.Properties[property]; ok {
				if s, ok := v.(string); ok {
					value = s
				}
			}
		}

		if value != "" {
			counts[value]++
		}
	}

	type item struct {
		Name  string
		Count int64
	}

	items := make([]item, 0, len(counts))
	for name, count := range counts {
		items = append(items, item{Name: name, Count: count})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	result := make([]map[string]interface{}, 0, len(items))
	total := float64(len(events))
	for _, it := range items {
		result = append(result, map[string]interface{}{
			"name":  it.Name,
			"count": it.Count,
			"percent": math.Round(float64(it.Count)/total*10000) / 100,
		})
	}

	return result
}

// ComputePercentiles computes statistical percentiles from a slice of values.
func (a *MetricsAggregator) ComputePercentiles(values []float64) map[string]float64 {
	if len(values) == 0 {
		return nil
	}

	sort.Float64s(values)

	n := float64(len(values))
	return map[string]float64{
		"min":  values[0],
		"max":  values[len(values)-1],
		"avg":  mean(values),
		"p50":  percentile(values, 0.50),
		"p90":  percentile(values, 0.90),
		"p95":  percentile(values, 0.95),
		"p99":  percentile(values, 0.99),
		"count": n,
	}
}

func (a *MetricsAggregator) granularityToDuration(granularity string) time.Duration {
	switch granularity {
	case "minute":
		return time.Minute
	case "hour":
		return time.Hour
	case "day":
		return 24 * time.Hour
	default:
		return time.Hour
	}
}

func (a *MetricsAggregator) bucketTime(t time.Time, interval time.Duration) time.Time {
	return t.Truncate(interval)
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	idx := p * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))

	if lower == upper {
		return sorted[lower]
	}

	fraction := idx - float64(lower)
	return sorted[lower] + (sorted[upper]-sorted[lower])*fraction
}

func generateMetricsID(tenantID string, windowStart time.Time, granularity string) string {
	return fmt.Sprintf("%s_%s_%s",
		tenantID,
		windowStart.Format("20060102_150405"),
		granularity,
	)
}
