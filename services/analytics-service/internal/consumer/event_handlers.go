package consumer

import (
	"context"
	"fmt"
	"log"
	"time"

	"analytics-service/internal/domain"
)

// EventHandler processes consumed Kafka events and stores them.
type EventHandler struct {
	eventRepo    domain.EventRepository
	metricsRepo  domain.MetricsRepository
	userRepo     domain.UserRepository
	batchSize    int
	batchTimeout time.Duration
	eventBuffer  []*domain.Event
	timer        *time.Timer
}

// NewEventHandler creates a new EventHandler with buffering.
func NewEventHandler(
	eventRepo domain.EventRepository,
	metricsRepo domain.MetricsRepository,
	userRepo domain.UserRepository,
	batchSize int,
	batchTimeout time.Duration,
) *EventHandler {
	return &EventHandler{
		eventRepo:    eventRepo,
		metricsRepo:  metricsRepo,
		userRepo:     userRepo,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		eventBuffer:  make([]*domain.Event, 0, batchSize),
		timer:        time.NewTimer(batchTimeout),
	}
}

// ProcessEvent handles a single event.
func (h *EventHandler) ProcessEvent(ctx context.Context, event *domain.Event) error {
	event.ProcessedAt = time.Now().UTC()

	// Enrich event with geo/device data if available.
	h.enrichEvent(event)

	// Buffer the event.
	h.eventBuffer = append(h.eventBuffer, event)

	// Flush if buffer is full.
	if len(h.eventBuffer) >= h.batchSize {
		if err := h.flush(ctx); err != nil {
			return err
		}
	}

	return nil
}

// ProcessBatch handles a batch of events.
func (h *EventHandler) ProcessBatch(ctx context.Context, events []*domain.Event) error {
	now := time.Now().UTC()

	for _, event := range events {
		event.ProcessedAt = now
		h.enrichEvent(event)
	}

	// Add to buffer.
	h.eventBuffer = append(h.eventBuffer, events...)

	// Flush if buffer reaches threshold.
	for len(h.eventBuffer) >= h.batchSize {
		if err := h.flush(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Flush writes all buffered events to storage.
func (h *EventHandler) Flush(ctx context.Context) error {
	return h.flush(ctx)
}

func (h *EventHandler) flush(ctx context.Context) error {
	if len(h.eventBuffer) == 0 {
		return nil
	}

	// Take a snapshot of the buffer.
	toFlush := make([]*domain.Event, len(h.eventBuffer))
	copy(toFlush, h.eventBuffer)
	h.eventBuffer = h.eventBuffer[:0]

	// Reset the flush timer.
	if !h.timer.Stop() {
		select {
		case <-h.timer.C:
		default:
		}
	}
	h.timer.Reset(h.batchTimeout)

	// Batch insert events.
	if err := h.eventRepo.BatchInsert(ctx, toFlush); err != nil {
		// On failure, put events back in the buffer for retry.
		h.eventBuffer = append(h.eventBuffer, toFlush...)
		return fmt.Errorf("failed to batch insert events: %w", err)
	}

	// Update user session tracking.
	h.trackSessions(ctx, toFlush)

	log.Printf("[event-handler] flushed %d events", len(toFlush))

	return nil
}

// RunFlushTimer periodically flushes the buffer on timeout.
func (h *EventHandler) RunFlushTimer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Final flush before exit.
			if err := h.Flush(ctx); err != nil {
				log.Printf("[event-handler] final flush error: %v", err)
			}
			return
		case <-h.timer.C:
			if err := h.Flush(ctx); err != nil {
				log.Printf("[event-handler] timer flush error: %v", err)
			}
			h.timer.Reset(h.batchTimeout)
		}
	}
}

func (h *EventHandler) enrichEvent(event *domain.Event) {
	// In production, enrich with:
	// - GeoIP lookup from IP address
	// - User agent parsing for device/OS/browser
	// - URL normalization
	// - Bot filtering

	if event.Properties == nil {
		event.Properties = make(map[string]interface{})
	}

	// Add processing metadata.
	event.Properties["processing_version"] = "1.0"
	event.Properties["processed_by"] = "analytics-service"
}

func (h *EventHandler) trackSessions(ctx context.Context, events []*domain.Event) {
	// Extract session data from events and update session tracking.
	sessionDurations := make(map[string]float64)
	sessionUsers := make(map[string]string)

	for _, e := range events {
		if e.SessionID == "" {
			continue
		}

		sessionUsers[e.SessionID] = e.UserID

		// Calculate session duration from event properties.
		if dur, ok := e.Properties["session_duration"]; ok {
			if d, ok := dur.(float64); ok {
				sessionDurations[e.SessionID] = d
			}
		}
	}

	// Persist session data.
	for sessionID, userID := range sessionUsers {
		duration := sessionDurations[sessionID]
		if err := h.userRepo.TrackSession(ctx, userID, sessionID, duration); err != nil {
			log.Printf("[event-handler] failed to track session %s: %v", sessionID, err)
		}
	}
}
