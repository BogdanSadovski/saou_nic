package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/real-ass/user-service/internal/domain"
)

type SubscriptionService struct {
	repo domain.UserRepository
}

func NewSubscriptionService(repo domain.UserRepository) *SubscriptionService {
	return &SubscriptionService{repo: repo}
}

// Define available plans
var SubscriptionPlans = map[domain.SubscriptionTier]*domain.SubscriptionPlan{
	domain.TierTrial: {
		ID:           "plan_trial",
		Tier:         domain.TierTrial,
		Name:         "Trial",
		Description:  "Free 14-day trial with core features",
		Price:        0,
		BillingCycle: "one-time",
		TrialDays:    14,
		Features: []string{
			"5 interviews per month",
			"Basic analytics",
			"Standard support",
			"Resume builder",
		},
		Limits: map[string]interface{}{
			"interviews_per_month": 5,
			"storage_gb":           1,
			"concurrent_sessions":  1,
		},
	},
	domain.TierPro: {
		ID:           "plan_pro",
		Tier:         domain.TierPro,
		Name:         "Pro",
		Description:  "Professional plan for serious candidates",
		Price:        9.99,
		BillingCycle: "monthly",
		Features: []string{
			"30 interviews per month",
			"Advanced analytics",
			"Priority support",
			"Resume builder",
			"Interview history",
			"Custom templates",
			"Performance insights",
		},
		Limits: map[string]interface{}{
			"interviews_per_month": 30,
			"storage_gb":           5,
			"concurrent_sessions":  3,
		},
	},
	domain.TierPlatinum: {
		ID:           "plan_platinum",
		Tier:         domain.TierPlatinum,
		Name:         "Platinum",
		Description:  "Ultimate plan with all features and priority",
		Price:        29.99,
		BillingCycle: "monthly",
		Features: []string{
			"Unlimited interviews",
			"Real-time analytics",
			"24/7 premium support",
			"Resume builder",
			"Interview history",
			"Custom templates",
			"Performance insights",
			"Team collaboration",
			"API access",
			"Custom branding",
		},
		Limits: map[string]interface{}{
			"interviews_per_month": 999999,
			"storage_gb":           100,
			"concurrent_sessions":  10,
		},
	},
}

// GetAvailablePlans returns all subscription plans
func (s *SubscriptionService) GetAvailablePlans(ctx context.Context) ([]*domain.SubscriptionPlan, error) {
	plans := make([]*domain.SubscriptionPlan, 0, len(SubscriptionPlans))
	for _, tier := range []domain.SubscriptionTier{domain.TierTrial, domain.TierPro, domain.TierPlatinum} {
		if plan, exists := SubscriptionPlans[tier]; exists {
			plans = append(plans, plan)
		}
	}
	return plans, nil
}

// GetPlanByTier returns a subscription plan by tier
func (s *SubscriptionService) GetPlanByTier(tier domain.SubscriptionTier) *domain.SubscriptionPlan {
	return SubscriptionPlans[tier]
}

// CreateSubscription creates a new subscription for user
func (s *SubscriptionService) CreateSubscription(ctx context.Context, userID uuid.UUID, req domain.CreateSubscriptionRequest) (*domain.Subscription, error) {
	// Validate tier
	if _, exists := SubscriptionPlans[req.Tier]; !exists {
		return nil, errors.New("invalid subscription tier")
	}

	// Check if user already has active subscription
	existingSub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err == nil && existingSub != nil {
		return nil, errors.New("user already has active subscription")
	}

	plan := SubscriptionPlans[req.Tier]

	now := time.Now().UTC()
	endDate := now.AddDate(0, 1, 0) // Default 1 month

	// If trial tier, use trial days
	if req.Tier == domain.TierTrial {
		trialEnd := now.AddDate(0, 0, plan.TrialDays)
		endDate = trialEnd
	}

	sub := &domain.Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Tier:      req.Tier,
		Status:    domain.SubscriptionActive,
		StartDate: now,
		EndDate:   endDate,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// For trial, set trial end date
	if req.Tier == domain.TierTrial {
		sub.TrialEndDate = &endDate
	} else {
		// For paid plans, set renewal date
		renewalDate := endDate
		sub.RenewalDate = &renewalDate
	}

	if req.PaymentMethodID != "" {
		sub.PaymentMethodID = req.PaymentMethodID
	}

	// Save subscription
	err = s.repo.CreateSubscription(ctx, sub)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

// GetUserSubscription returns current active subscription for user
func (s *SubscriptionService) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return s.repo.GetActiveSubscription(ctx, userID)
}

// CancelSubscription cancels user's subscription
func (s *SubscriptionService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return errors.New("no active subscription found")
	}

	now := time.Now().UTC()
	sub.Status = domain.SubscriptionCanceled
	sub.IsActive = false
	sub.CanceledAt = &now
	sub.UpdatedAt = now

	return s.repo.UpdateSubscription(ctx, sub)
}

// UpgradeSubscription upgrades user's subscription to a higher tier
func (s *SubscriptionService) UpgradeSubscription(ctx context.Context, userID uuid.UUID, newTier domain.SubscriptionTier) (*domain.Subscription, error) {
	if _, exists := SubscriptionPlans[newTier]; !exists {
		return nil, errors.New("invalid subscription tier")
	}

	currentSub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if currentSub == nil {
		return nil, errors.New("no active subscription found")
	}

	// Cancel old subscription
	now := time.Now().UTC()
	currentSub.Status = domain.SubscriptionCanceled
	currentSub.IsActive = false
	currentSub.CanceledAt = &now
	currentSub.UpdatedAt = now
	if err := s.repo.UpdateSubscription(ctx, currentSub); err != nil {
		return nil, err
	}

	// Create new subscription
	newSub := &domain.Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Tier:      newTier,
		Status:    domain.SubscriptionActive,
		StartDate: now,
		EndDate:   now.AddDate(0, 1, 0),
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	plan := SubscriptionPlans[newTier]
	if newTier == domain.TierTrial {
		trialEnd := now.AddDate(0, 0, plan.TrialDays)
		newSub.EndDate = trialEnd
		newSub.TrialEndDate = &trialEnd
	} else {
		renewalDate := newSub.EndDate
		newSub.RenewalDate = &renewalDate
	}

	if err := s.repo.CreateSubscription(ctx, newSub); err != nil {
		return nil, err
	}

	return newSub, nil
}

// CheckSubscriptionExpiry checks if subscription has expired
func (s *SubscriptionService) CheckSubscriptionExpiry(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return errors.New("no active subscription found")
	}

	now := time.Now().UTC()
	if now.After(sub.EndDate) {
		sub.Status = domain.SubscriptionExpired
		sub.IsActive = false
		sub.UpdatedAt = now
		return s.repo.UpdateSubscription(ctx, sub)
	}

	return nil
}
