package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/real-ass/user-service/internal/domain"
)

func scanSubscription(scanner interface {
	Scan(dest ...interface{}) error
}, sub *domain.Subscription) error {
	return scanner.Scan(
		&sub.ID,
		&sub.UserID,
		&sub.Tier,
		&sub.Status,
		&sub.StartDate,
		&sub.EndDate,
		&sub.RenewalDate,
		&sub.TrialEndDate,
		&sub.PaymentMethodID,
		&sub.IsActive,
		&sub.CreatedAt,
		&sub.UpdatedAt,
		&sub.CanceledAt,
	)
}

func scanPaymentIntent(scanner interface {
	Scan(dest ...interface{}) error
}, intent *domain.PaymentIntent) error {
	return scanner.Scan(
		&intent.ID,
		&intent.UserID,
		&intent.Tier,
		&intent.BillingCycle,
		&intent.AmountCents,
		&intent.Currency,
		&intent.Status,
		&intent.Provider,
		&intent.ClientSecret,
		&intent.PaymentMethodID,
		&intent.PromoCodeID,
		&intent.ExpiresAt,
		&intent.ConfirmedAt,
		&intent.CreatedAt,
		&intent.UpdatedAt,
	)
}

func scanPaymentTransaction(scanner interface {
	Scan(dest ...interface{}) error
}, tx *domain.PaymentTransaction) error {
	return scanner.Scan(
		&tx.ID,
		&tx.IntentID,
		&tx.UserID,
		&tx.SubscriptionID,
		&tx.AmountCents,
		&tx.Currency,
		&tx.Status,
		&tx.Provider,
		&tx.ExternalReference,
		&tx.Description,
		&tx.CreatedAt,
	)
}

func (r *postgresRepository) CreateSubscription(ctx context.Context, sub *domain.Subscription) error {
	_, err := r.pool.Exec(ctx, queryCreateSubscription,
		sub.ID,
		sub.UserID,
		sub.Tier,
		sub.Status,
		sub.StartDate,
		sub.EndDate,
		sub.RenewalDate,
		sub.TrialEndDate,
		sub.PaymentMethodID,
		sub.IsActive,
		sub.CreatedAt,
		sub.UpdatedAt,
		sub.CanceledAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetActiveSubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	sub := &domain.Subscription{}
	err := scanSubscription(r.pool.QueryRow(ctx, queryGetActiveSubscription, userID), sub)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active subscription: %w", err)
	}
	return sub, nil
}

func (r *postgresRepository) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	sub := &domain.Subscription{}
	err := scanSubscription(r.pool.QueryRow(ctx, queryGetSubscriptionByID, id), sub)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subscription by id: %w", err)
	}
	return sub, nil
}

func (r *postgresRepository) UpdateSubscription(ctx context.Context, sub *domain.Subscription) error {
	_, err := r.pool.Exec(ctx, queryUpdateSubscription,
		sub.ID,
		sub.Tier,
		sub.Status,
		sub.StartDate,
		sub.EndDate,
		sub.RenewalDate,
		sub.TrialEndDate,
		sub.PaymentMethodID,
		sub.IsActive,
		sub.UpdatedAt,
		sub.CanceledAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	return nil
}

func (r *postgresRepository) ListSubscriptions(ctx context.Context, userID uuid.UUID) ([]*domain.Subscription, error) {
	rows, err := r.pool.Query(ctx, queryListSubscriptions, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.Subscription, 0)
	for rows.Next() {
		sub := &domain.Subscription{}
		if err := scanSubscription(rows, sub); err != nil {
			return nil, fmt.Errorf("failed to scan subscription: %w", err)
		}
		result = append(result, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating subscriptions: %w", err)
	}
	return result, nil
}

func (r *postgresRepository) CreatePaymentIntent(ctx context.Context, intent *domain.PaymentIntent) error {
	_, err := r.pool.Exec(ctx, queryCreatePaymentIntent,
		intent.ID,
		intent.UserID,
		intent.Tier,
		intent.BillingCycle,
		intent.AmountCents,
		intent.Currency,
		intent.Status,
		intent.Provider,
		intent.ClientSecret,
		intent.PaymentMethodID,
		intent.PromoCodeID,
		intent.ExpiresAt,
		intent.ConfirmedAt,
		intent.CreatedAt,
		intent.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create payment intent: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetPaymentIntentByID(ctx context.Context, id uuid.UUID) (*domain.PaymentIntent, error) {
	intent := &domain.PaymentIntent{}
	err := scanPaymentIntent(r.pool.QueryRow(ctx, queryGetPaymentIntentByID, id), intent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment intent: %w", err)
	}
	return intent, nil
}

func (r *postgresRepository) UpdatePaymentIntent(ctx context.Context, intent *domain.PaymentIntent) error {
	_, err := r.pool.Exec(ctx, queryUpdatePaymentIntent,
		intent.ID,
		intent.Status,
		intent.PaymentMethodID,
		intent.PromoCodeID,
		intent.ExpiresAt,
		intent.ConfirmedAt,
		intent.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update payment intent: %w", err)
	}
	return nil
}

func (r *postgresRepository) CreatePaymentTransaction(ctx context.Context, tx *domain.PaymentTransaction) error {
	_, err := r.pool.Exec(ctx, queryCreatePaymentTransaction,
		tx.ID,
		tx.IntentID,
		tx.UserID,
		tx.SubscriptionID,
		tx.AmountCents,
		tx.Currency,
		tx.Status,
		tx.Provider,
		tx.ExternalReference,
		tx.Description,
		tx.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create payment transaction: %w", err)
	}
	return nil
}

func (r *postgresRepository) ListPaymentTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.PaymentTransaction, error) {
	rows, err := r.pool.Query(ctx, queryListPaymentTransactions, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment transactions: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.PaymentTransaction, 0)
	for rows.Next() {
		tx := &domain.PaymentTransaction{}
		if err := scanPaymentTransaction(rows, tx); err != nil {
			return nil, fmt.Errorf("failed to scan payment transaction: %w", err)
		}
		result = append(result, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating payment transactions: %w", err)
	}
	return result, nil
}
