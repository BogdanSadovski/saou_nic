package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/real-ass/user-service/internal/domain"
)

var (
	ErrIntentNotFound      = errors.New("checkout intent not found")
	ErrIntentWrongState    = errors.New("checkout intent has invalid status")
	ErrTrialDoesNotRequire = errors.New("trial tier does not require payment")
)

type PaymentService struct {
	repo              domain.UserRepository
	subscriptionStore *SubscriptionService
}

func NewPaymentService(repo domain.UserRepository, subscriptionStore *SubscriptionService) *PaymentService {
	return &PaymentService{repo: repo, subscriptionStore: subscriptionStore}
}

func (s *PaymentService) CreateCheckoutIntent(ctx context.Context, userID uuid.UUID, req domain.CreateCheckoutIntentRequest) (*domain.PaymentIntent, error) {
	plan := s.subscriptionStore.GetPlanByTier(req.Tier)
	if plan == nil {
		return nil, errors.New("invalid subscription tier")
	}

	if req.Tier == domain.TierTrial {
		return nil, ErrTrialDoesNotRequire
	}

	billingCycle := req.BillingCycle
	if billingCycle == "" {
		billingCycle = plan.BillingCycle
	}

	now := time.Now().UTC()
	intent := &domain.PaymentIntent{
		ID:              uuid.New(),
		UserID:          userID,
		Tier:            req.Tier,
		BillingCycle:    billingCycle,
		AmountCents:     int64(math.Round(plan.Price * 100)),
		Currency:        "USD",
		Status:          domain.PaymentIntentRequiresConfirmation,
		Provider:        "test-gateway",
		ClientSecret:    fmt.Sprintf("test_secret_%s", uuid.NewString()),
		PaymentMethodID: req.PaymentMethodID,
		PromoCodeID:     req.PromoCodeID,
		ExpiresAt:       now.Add(30 * time.Minute),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreatePaymentIntent(ctx, intent); err != nil {
		return nil, err
	}

	return intent, nil
}

func (s *PaymentService) ConfirmCheckoutIntent(ctx context.Context, userID, intentID uuid.UUID) (*domain.PaymentIntent, *domain.Subscription, *domain.PaymentTransaction, error) {
	intent, err := s.repo.GetPaymentIntentByID(ctx, intentID)
	if err != nil {
		return nil, nil, nil, err
	}
	if intent == nil || intent.UserID != userID {
		return nil, nil, nil, ErrIntentNotFound
	}
	if intent.Status != domain.PaymentIntentRequiresConfirmation {
		return nil, nil, nil, ErrIntentWrongState
	}

	now := time.Now().UTC()
	intent.Status = domain.PaymentIntentSucceeded
	intent.ConfirmedAt = &now
	intent.UpdatedAt = now
	if err := s.repo.UpdatePaymentIntent(ctx, intent); err != nil {
		return nil, nil, nil, err
	}

	sub, err := s.activateSubscriptionForIntent(ctx, intent, now)
	if err != nil {
		return nil, nil, nil, err
	}

	tx := &domain.PaymentTransaction{
		ID:                uuid.New(),
		IntentID:          &intent.ID,
		UserID:            userID,
		SubscriptionID:    &sub.ID,
		AmountCents:       intent.AmountCents,
		Currency:          intent.Currency,
		Status:            domain.PaymentTransactionSucceeded,
		Provider:          intent.Provider,
		ExternalReference: fmt.Sprintf("txn_%s", uuid.NewString()),
		Description:       fmt.Sprintf("%s subscription payment (%s)", intent.Tier, intent.BillingCycle),
		CreatedAt:         now,
	}
	if err := s.repo.CreatePaymentTransaction(ctx, tx); err != nil {
		return nil, nil, nil, err
	}

	return intent, sub, tx, nil
}

func (s *PaymentService) ApplyTestWebhook(ctx context.Context, userID, intentID uuid.UUID, status string) (*domain.PaymentIntent, error) {
	intent, err := s.repo.GetPaymentIntentByID(ctx, intentID)
	if err != nil {
		return nil, err
	}
	if intent == nil || intent.UserID != userID {
		return nil, ErrIntentNotFound
	}

	now := time.Now().UTC()
	switch status {
	case string(domain.PaymentIntentSucceeded):
		_, _, _, err := s.ConfirmCheckoutIntent(ctx, userID, intentID)
		if err != nil {
			return nil, err
		}
		return s.repo.GetPaymentIntentByID(ctx, intentID)
	case string(domain.PaymentIntentFailed):
		intent.Status = domain.PaymentIntentFailed
	case string(domain.PaymentIntentCanceled):
		intent.Status = domain.PaymentIntentCanceled
	default:
		return nil, errors.New("unsupported webhook status")
	}

	intent.UpdatedAt = now
	if err := s.repo.UpdatePaymentIntent(ctx, intent); err != nil {
		return nil, err
	}

	if intent.Status == domain.PaymentIntentFailed {
		tx := &domain.PaymentTransaction{
			ID:                uuid.New(),
			IntentID:          &intent.ID,
			UserID:            intent.UserID,
			AmountCents:       intent.AmountCents,
			Currency:          intent.Currency,
			Status:            domain.PaymentTransactionFailed,
			Provider:          intent.Provider,
			ExternalReference: fmt.Sprintf("txn_fail_%s", uuid.NewString()),
			Description:       "Test gateway declined payment",
			CreatedAt:         now,
		}
		if err := s.repo.CreatePaymentTransaction(ctx, tx); err != nil {
			return nil, err
		}
	}

	return intent, nil
}

func (s *PaymentService) ListTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.PaymentTransaction, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListPaymentTransactions(ctx, userID, limit)
}

func (s *PaymentService) activateSubscriptionForIntent(ctx context.Context, intent *domain.PaymentIntent, now time.Time) (*domain.Subscription, error) {
	existing, err := s.repo.GetActiveSubscription(ctx, intent.UserID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		existing.Status = domain.SubscriptionCanceled
		existing.IsActive = false
		existing.CanceledAt = &now
		existing.UpdatedAt = now
		if err := s.repo.UpdateSubscription(ctx, existing); err != nil {
			return nil, err
		}
	}

	endDate := now.AddDate(0, 1, 0)
	renewalDate := endDate
	sub := &domain.Subscription{
		ID:              uuid.New(),
		UserID:          intent.UserID,
		Tier:            intent.Tier,
		Status:          domain.SubscriptionActive,
		StartDate:       now,
		EndDate:         endDate,
		RenewalDate:     &renewalDate,
		PaymentMethodID: intent.PaymentMethodID,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}
