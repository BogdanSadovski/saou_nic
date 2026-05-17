package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/real-ass/admin-service/internal/domain"
)

// SubscriptionService handles subscription management operations.
type SubscriptionService struct {
	subRepo   domain.SubscriptionRepository
	userRepo  domain.UserRepository
	auditRepo domain.AuditLogRepository
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	subRepo domain.SubscriptionRepository,
	userRepo domain.UserRepository,
	auditRepo domain.AuditLogRepository,
) *SubscriptionService {
	return &SubscriptionService{
		subRepo:   subRepo,
		userRepo:  userRepo,
		auditRepo: auditRepo,
	}
}

// CreateSubscription creates a new subscription for a user.
func (s *SubscriptionService) CreateSubscription(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier, adminID uuid.UUID) (*domain.Subscription, error) {
	// Verify user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user already has an active subscription
	existing, err := s.subRepo.GetByUserID(ctx, userID)
	if err == nil && existing != nil && existing.Status == domain.SubscriptionActive {
		return nil, fmt.Errorf("user already has an active subscription")
	}

	now := time.Now()
	endDate := now.AddDate(0, 1, 0) // 1 month default

	subscription := &domain.Subscription{
		ID:           uuid.New(),
		UserID:       userID,
		Tier:         tier,
		Status:       domain.SubscriptionActive,
		StartDate:    now,
		EndDate:      &endDate,
		AutoRenew:    true,
		MaxUsers:     tierMaxUsers(tier),
		MaxStorageGB: tierMaxStorage(tier),
		Features:     tierFeatures(tier),
		Metadata:     make(map[string]string),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.subRepo.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Log the action
	s.logAudit(ctx, adminID, user.Email, domain.ActionCreate, "subscription", &subscription.ID,
		fmt.Sprintf("Created %s subscription for user %s", tier, user.Email), "")

	return subscription, nil
}

// GetSubscription retrieves a subscription by ID.
func (s *SubscriptionService) GetSubscription(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	return sub, nil
}

// GetSubscriptionByUserID retrieves a user's subscription.
func (s *SubscriptionService) GetSubscriptionByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	return sub, nil
}

// UpdateSubscription updates an existing subscription.
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, id uuid.UUID, tier domain.SubscriptionTier, autoRenew bool, adminID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	user, _ := s.userRepo.GetByID(ctx, sub.UserID)
	oldTier := sub.Tier

	sub.Tier = tier
	sub.AutoRenew = autoRenew
	sub.MaxUsers = tierMaxUsers(tier)
	sub.MaxStorageGB = tierMaxStorage(tier)
	sub.Features = tierFeatures(tier)
	sub.UpdatedAt = time.Now()

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Log the action
	adminEmail := ""
	if user != nil {
		adminEmail = user.Email
	}
	s.logAudit(ctx, adminID, adminEmail, domain.ActionChangeSubscription, "subscription", &sub.ID,
		fmt.Sprintf("Updated subscription for user %s from %s to %s", sub.UserID, oldTier, tier), "")

	return sub, nil
}

// CancelSubscription cancels an active subscription.
func (s *SubscriptionService) CancelSubscription(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	sub, err := s.subRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub.Status != domain.SubscriptionActive {
		return fmt.Errorf("subscription is not active")
	}

	sub.Status = domain.SubscriptionCanceled
	sub.UpdatedAt = time.Now()

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	user, _ := s.userRepo.GetByID(ctx, sub.UserID)
	adminEmail := ""
	if user != nil {
		adminEmail = user.Email
	}
	s.logAudit(ctx, adminID, adminEmail, domain.ActionChangeSubscription, "subscription", &sub.ID,
		fmt.Sprintf("Canceled subscription for user %s", sub.UserID), "")

	return nil
}

// RenewSubscription renews an expired subscription.
func (s *SubscriptionService) RenewSubscription(ctx context.Context, id uuid.UUID, adminID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub.Status != domain.SubscriptionExpired && sub.Status != domain.SubscriptionCanceled {
		return nil, fmt.Errorf("subscription cannot be renewed in current status: %s", sub.Status)
	}

	now := time.Now()
	newEndDate := now.AddDate(0, 1, 0)

	sub.Status = domain.SubscriptionActive
	sub.StartDate = now
	sub.EndDate = &newEndDate
	sub.UpdatedAt = now

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to renew subscription: %w", err)
	}

	user, _ := s.userRepo.GetByID(ctx, sub.UserID)
	adminEmail := ""
	if user != nil {
		adminEmail = user.Email
	}
	s.logAudit(ctx, adminID, adminEmail, domain.ActionChangeSubscription, "subscription", &sub.ID,
		fmt.Sprintf("Renewed subscription for user %s", sub.UserID), "")

	return sub, nil
}

// ListSubscriptionsByStatus lists all subscriptions with a given status.
func (s *SubscriptionService) ListSubscriptionsByStatus(ctx context.Context, status domain.SubscriptionStatus) ([]domain.Subscription, error) {
	subs, err := s.subRepo.ListByStatus(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	return subs, nil
}

// ListSubscriptionsByTier lists all subscriptions with a given tier.
func (s *SubscriptionService) ListSubscriptionsByTier(ctx context.Context, tier domain.SubscriptionTier) ([]domain.Subscription, error) {
	subs, err := s.subRepo.ListByTier(ctx, tier)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	return subs, nil
}

// ExpireOldSubscriptions expires subscriptions that have passed their end date.
func (s *SubscriptionService) ExpireOldSubscriptions(ctx context.Context, adminID uuid.UUID) (int64, error) {
	expired, err := s.subRepo.ExpireOldSubscriptions(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to expire old subscriptions: %w", err)
	}

	if expired > 0 {
		s.logAudit(ctx, adminID, "system", domain.ActionChangeSubscription, "subscription", nil,
			fmt.Sprintf("Auto-expired %d subscriptions", expired), "")
	}

	return expired, nil
}

func (s *SubscriptionService) logAudit(ctx context.Context, adminID uuid.UUID, adminEmail string,
	action domain.AuditAction, resourceType string, resourceID *uuid.UUID, details, ipAddress string) {

	log := &domain.AuditLog{
		ID:           uuid.New(),
		AdminID:      adminID,
		AdminEmail:   adminEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		CreatedAt:    time.Now(),
	}

	go func() {
		_ = s.auditRepo.Create(context.Background(), log)
	}()
}

// tierMonthlyPriceUSD returns the published monthly subscription
// price for each tier in US dollars. Used by both Subscription
// records (sub.Amount) and dashboard revenue aggregation.
//
// Keep in sync with the frontend TIER_CATALOG in
// app/store/subscriptionStore.ts so admin numbers match what the
// user sees on the checkout page.
func tierMonthlyPriceUSD(tier domain.SubscriptionTier) float64 {
	switch tier {
	case domain.TierBasic, "starter":
		return 9
	case domain.TierPro:
		return 19
	case domain.TierEnterprise, "team":
		return 49
	default:
		return 0
	}
}

// Tier configuration helpers
func tierMaxUsers(tier domain.SubscriptionTier) int {
	switch tier {
	case domain.TierFree:
		return 1
	case domain.TierBasic:
		return 5
	case domain.TierPro:
		return 25
	case domain.TierEnterprise:
		return -1 // Unlimited
	default:
		return 1
	}
}

func tierMaxStorage(tier domain.SubscriptionTier) int {
	switch tier {
	case domain.TierFree:
		return 1
	case domain.TierBasic:
		return 10
	case domain.TierPro:
		return 100
	case domain.TierEnterprise:
		return -1 // Unlimited
	default:
		return 1
	}
}

func tierFeatures(tier domain.SubscriptionTier) []string {
	switch tier {
	case domain.TierFree:
		return []string{"basic_access"}
	case domain.TierBasic:
		return []string{"basic_access", "priority_support", "custom_branding"}
	case domain.TierPro:
		return []string{"basic_access", "priority_support", "custom_branding", "api_access", "analytics"}
	case domain.TierEnterprise:
		return []string{"basic_access", "priority_support", "custom_branding", "api_access", "analytics", "sso", "dedicated_support", "sla"}
	default:
		return []string{}
	}
}
