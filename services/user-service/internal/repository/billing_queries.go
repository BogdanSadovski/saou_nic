package repository

const (
	queryCreateSubscription = `
		INSERT INTO subscriptions (
			id, user_id, tier, status, start_date, end_date, renewal_date, trial_end_date,
			payment_method_id, is_active, created_at, updated_at, canceled_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13
		)
	`

	queryGetActiveSubscription = `
		SELECT id, user_id, tier, status, start_date, end_date, renewal_date, trial_end_date,
		       payment_method_id, is_active, created_at, updated_at, canceled_at
		FROM subscriptions
		WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`

	queryGetSubscriptionByID = `
		SELECT id, user_id, tier, status, start_date, end_date, renewal_date, trial_end_date,
		       payment_method_id, is_active, created_at, updated_at, canceled_at
		FROM subscriptions
		WHERE id = $1
	`

	queryUpdateSubscription = `
		UPDATE subscriptions
		SET tier = $2,
		    status = $3,
		    start_date = $4,
		    end_date = $5,
		    renewal_date = $6,
		    trial_end_date = $7,
		    payment_method_id = $8,
		    is_active = $9,
		    updated_at = $10,
		    canceled_at = $11
		WHERE id = $1
	`

	queryListSubscriptions = `
		SELECT id, user_id, tier, status, start_date, end_date, renewal_date, trial_end_date,
		       payment_method_id, is_active, created_at, updated_at, canceled_at
		FROM subscriptions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	queryCreatePaymentIntent = `
		INSERT INTO payment_intents (
			id, user_id, tier, billing_cycle, amount_cents, currency, status,
			provider, client_secret, payment_method_id, promo_code_id,
			expires_at, confirmed_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15
		)
	`

	queryGetPaymentIntentByID = `
		SELECT id, user_id, tier, billing_cycle, amount_cents, currency, status,
		       provider, client_secret, payment_method_id, promo_code_id,
		       expires_at, confirmed_at, created_at, updated_at
		FROM payment_intents
		WHERE id = $1
	`

	queryUpdatePaymentIntent = `
		UPDATE payment_intents
		SET status = $2,
		    payment_method_id = $3,
		    promo_code_id = $4,
		    expires_at = $5,
		    confirmed_at = $6,
		    updated_at = $7
		WHERE id = $1
	`

	queryCreatePaymentTransaction = `
		INSERT INTO payment_transactions (
			id, intent_id, user_id, subscription_id, amount_cents, currency,
			status, provider, external_reference, description, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11
		)
	`

	queryListPaymentTransactions = `
		SELECT id, intent_id, user_id, subscription_id, amount_cents, currency,
		       status, provider, external_reference, description, created_at
		FROM payment_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
)
