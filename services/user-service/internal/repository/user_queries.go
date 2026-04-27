package repository

const (
	queryGetUserByID = `
		SELECT id, email, username, password_hash, first_name, last_name,
		       avatar_url, role, status, provider, provider_id, email_verified,
		       created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`

	queryGetUserByEmail = `
		SELECT id, email, username, password_hash, first_name, last_name,
		       avatar_url, role, status, provider, provider_id, email_verified,
		       created_at, updated_at, last_login_at
		FROM users WHERE email = $1
	`

	queryGetUserByUsername = `
		SELECT id, email, username, password_hash, first_name, last_name,
		       avatar_url, role, status, provider, provider_id, email_verified,
		       created_at, updated_at, last_login_at
		FROM users WHERE username = $1
	`

	queryGetUserByProviderID = `
		SELECT id, email, username, password_hash, first_name, last_name,
		       avatar_url, role, status, provider, provider_id, email_verified,
		       created_at, updated_at, last_login_at
		FROM users WHERE provider = $1 AND provider_id = $2
	`

	queryUpdateLastLogin = `
		UPDATE users SET last_login_at = $1 WHERE id = $2
	`

	queryDeleteUser = `
		DELETE FROM users WHERE id = $1
	`

	queryListUsers = `
		SELECT id, email, username, password_hash, first_name, last_name,
		       avatar_url, role, status, provider, provider_id, email_verified,
		       created_at, updated_at, last_login_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`

	queryCountUsers = `
		SELECT COUNT(*) FROM users
	`
)
