package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/real-ass/admin-service/internal/config"
	"github.com/real-ass/admin-service/internal/domain"
)

// PostgresRepository implements all repository interfaces using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository.
func NewPostgresRepository(ctx context.Context, cfg *config.DatabaseConfig) (*PostgresRepository, error) {
	dbConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	dbConfig.MaxConns = cfg.MaxConns
	dbConfig.MinConns = cfg.MinConns
	dbConfig.MaxConnLifetime = cfg.MaxConnLifetime
	dbConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresRepository{pool: pool}, nil
}

// Close closes the database connection pool.
func (r *PostgresRepository) Close() {
	r.pool.Close()
}

// Pool returns the underlying connection pool.
func (r *PostgresRepository) Pool() *pgxpool.Pool {
	return r.pool
}

// ==================== User Repository ====================

func (r *PostgresRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, username, password_hash, role, status, first_name, last_name,
		                   avatar_url, email_verified, two_factor_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.Role,
		user.Status,
		user.FirstName,
		user.LastName,
		user.AvatarURL,
		user.EmailVerified,
		user.TwoFactorEnabled,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, status, first_name, last_name,
		       avatar_url, last_login_at, email_verified, two_factor_enabled, created_at, updated_at, deleted_at
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
		&user.LastLoginAt,
		&user.EmailVerified,
		&user.TwoFactorEnabled,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, err
}

func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, status, first_name, last_name,
		       avatar_url, last_login_at, email_verified, two_factor_enabled, created_at, updated_at, deleted_at
		FROM users WHERE email = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
		&user.LastLoginAt,
		&user.EmailVerified,
		&user.TwoFactorEnabled,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, err
}

func (r *PostgresRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, status, first_name, last_name,
		       avatar_url, last_login_at, email_verified, two_factor_enabled, created_at, updated_at, deleted_at
		FROM users WHERE username = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
		&user.LastLoginAt,
		&user.EmailVerified,
		&user.TwoFactorEnabled,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, err
}

func (r *PostgresRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users SET email = $1, username = $2, role = $3, status = $4,
		                 first_name = $5, last_name = $6, avatar_url = $7,
		                 email_verified = $8, two_factor_enabled = $9, updated_at = $10
		WHERE id = $11 AND deleted_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query,
		user.Email,
		user.Username,
		user.Role,
		user.Status,
		user.FirstName,
		user.LastName,
		user.AvatarURL,
		user.EmailVerified,
		user.TwoFactorEnabled,
		user.UpdatedAt,
		user.ID,
	)

	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET deleted_at = $1, updated_at = $1 WHERE id = $2`
	now := time.Now()

	_, err := r.pool.Exec(ctx, query, now, id)
	return err
}

func (r *PostgresRepository) List(ctx context.Context, query domain.ListUsersQuery) ([]domain.User, int64, error) {
	baseQuery := `
		FROM users WHERE deleted_at IS NULL
	`

	args := make([]interface{}, 0)
	argIdx := 1

	if query.Role != nil {
		baseQuery += fmt.Sprintf(" AND role = $%d", argIdx)
		args = append(args, *query.Role)
		argIdx++
	}

	if query.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *query.Status)
		argIdx++
	}

	if query.Search != "" {
		baseQuery += fmt.Sprintf(" AND (email ILIKE $%d OR username ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+query.Search+"%")
		argIdx++
	}

	// Count query
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Data query with pagination and sorting
	offset := (query.Page - 1) * query.PageSize
	dataQuery := "SELECT id, email, username, password_hash, role, status, first_name, last_name, " +
		"avatar_url, last_login_at, email_verified, two_factor_enabled, created_at, updated_at, deleted_at " +
		baseQuery

	allowedSortFields := map[string]bool{
		"created_at":    true,
		"updated_at":    true,
		"email":         true,
		"username":      true,
		"last_login_at": true,
	}

	sortBy := "created_at"
	if query.SortBy != "" && allowedSortFields[query.SortBy] {
		sortBy = query.SortBy
	}

	order := "DESC"
	if query.Order == "asc" {
		order = "ASC"
	}

	dataQuery += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sortBy, order, argIdx, argIdx+1)
	args = append(args, query.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.PasswordHash,
			&user.Role,
			&user.Status,
			&user.FirstName,
			&user.LastName,
			&user.AvatarURL,
			&user.LastLoginAt,
			&user.EmailVerified,
			&user.TwoFactorEnabled,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

func (r *PostgresRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&count)
	return count, err
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	query := `UPDATE users SET status = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, status, time.Now(), id)
	return err
}

func (r *PostgresRepository) UpdateRole(ctx context.Context, id uuid.UUID, role domain.UserRole) error {
	query := `UPDATE users SET role = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, role, time.Now(), id)
	return err
}

func (r *PostgresRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL`
	now := time.Now()
	_, err := r.pool.Exec(ctx, query, now, now, id)
	return err
}

// ==================== Subscription Repository ====================

func (r *PostgresRepository) CreateSubscription(ctx context.Context, subscription *domain.Subscription) error {
	query := `
		INSERT INTO subscriptions (id, user_id, tier, status, start_date, end_date, auto_renew,
		                           max_users, max_storage_gb, features, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	features := subscription.Features
	if features == nil {
		features = []string{}
	}

	metadata := subscription.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}

	_, err := r.pool.Exec(ctx, query,
		subscription.ID,
		subscription.UserID,
		subscription.Tier,
		subscription.Status,
		subscription.StartDate,
		subscription.EndDate,
		subscription.AutoRenew,
		subscription.MaxUsers,
		subscription.MaxStorageGB,
		features,
		metadata,
		subscription.CreatedAt,
		subscription.UpdatedAt,
	)

	return err
}

func (r *PostgresRepository) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	query := `
		SELECT id, user_id, tier, status, start_date, end_date, auto_renew,
		       max_users, max_storage_gb, features, metadata, created_at, updated_at
		FROM subscriptions WHERE id = $1
	`

	sub := &domain.Subscription{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.Tier,
		&sub.Status,
		&sub.StartDate,
		&sub.EndDate,
		&sub.AutoRenew,
		&sub.MaxUsers,
		&sub.MaxStorageGB,
		&sub.Features,
		&sub.Metadata,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	return sub, err
}

func (r *PostgresRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	query := `
		SELECT id, user_id, tier, status, start_date, end_date, auto_renew,
		       max_users, max_storage_gb, features, metadata, created_at, updated_at
		FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1
	`

	sub := &domain.Subscription{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.Tier,
		&sub.Status,
		&sub.StartDate,
		&sub.EndDate,
		&sub.AutoRenew,
		&sub.MaxUsers,
		&sub.MaxStorageGB,
		&sub.Features,
		&sub.Metadata,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("subscription not found for user: %w", err)
	}

	return sub, err
}

func (r *PostgresRepository) UpdateSubscription(ctx context.Context, subscription *domain.Subscription) error {
	query := `
		UPDATE subscriptions SET tier = $1, status = $2, start_date = $3, end_date = $4,
		                         auto_renew = $5, max_users = $6, max_storage_gb = $7,
		                         features = $8, metadata = $9, updated_at = $10
		WHERE id = $11
	`

	_, err := r.pool.Exec(ctx, query,
		subscription.Tier,
		subscription.Status,
		subscription.StartDate,
		subscription.EndDate,
		subscription.AutoRenew,
		subscription.MaxUsers,
		subscription.MaxStorageGB,
		subscription.Features,
		subscription.Metadata,
		subscription.UpdatedAt,
		subscription.ID,
	)

	return err
}

func (r *PostgresRepository) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", id)
	return err
}

func (r *PostgresRepository) ListByStatus(ctx context.Context, status domain.SubscriptionStatus) ([]domain.Subscription, error) {
	query := `
		SELECT id, user_id, tier, status, start_date, end_date, auto_renew,
		       max_users, max_storage_gb, features, metadata, created_at, updated_at
		FROM subscriptions WHERE status = $1 ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []domain.Subscription
	for rows.Next() {
		var sub domain.Subscription
		if err := rows.Scan(
			&sub.ID,
			&sub.UserID,
			&sub.Tier,
			&sub.Status,
			&sub.StartDate,
			&sub.EndDate,
			&sub.AutoRenew,
			&sub.MaxUsers,
			&sub.MaxStorageGB,
			&sub.Features,
			&sub.Metadata,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

func (r *PostgresRepository) ListByTier(ctx context.Context, tier domain.SubscriptionTier) ([]domain.Subscription, error) {
	query := `
		SELECT id, user_id, tier, status, start_date, end_date, auto_renew,
		       max_users, max_storage_gb, features, metadata, created_at, updated_at
		FROM subscriptions WHERE tier = $1 ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []domain.Subscription
	for rows.Next() {
		var sub domain.Subscription
		if err := rows.Scan(
			&sub.ID,
			&sub.UserID,
			&sub.Tier,
			&sub.Status,
			&sub.StartDate,
			&sub.EndDate,
			&sub.AutoRenew,
			&sub.MaxUsers,
			&sub.MaxStorageGB,
			&sub.Features,
			&sub.Metadata,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

func (r *PostgresRepository) ExpireOldSubscriptions(ctx context.Context) (int64, error) {
	query := `
		UPDATE subscriptions SET status = 'expired', updated_at = $1
		WHERE status = 'active' AND end_date IS NOT NULL AND end_date < $1
	`

	result, err := r.pool.Exec(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// ==================== Audit Log Repository ====================

func (r *PostgresRepository) CreateAuditLog(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, admin_id, admin_email, action, resource_type, resource_id,
		                        details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		log.ID,
		log.AdminID,
		log.AdminEmail,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		log.Details,
		log.IPAddress,
		log.UserAgent,
		log.CreatedAt,
	)

	return err
}

func (r *PostgresRepository) GetAuditLogByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	query := `
		SELECT id, admin_id, admin_email, action, resource_type, resource_id,
		       details, ip_address, user_agent, created_at
		FROM audit_logs WHERE id = $1
	`

	log := &domain.AuditLog{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&log.ID,
		&log.AdminID,
		&log.AdminEmail,
		&log.Action,
		&log.ResourceType,
		&log.ResourceID,
		&log.Details,
		&log.IPAddress,
		&log.UserAgent,
		&log.CreatedAt,
	)

	return log, err
}

func (r *PostgresRepository) ListAuditLogs(ctx context.Context, filters domain.AuditLogFilters) ([]domain.AuditLog, int64, error) {
	baseQuery := `FROM audit_logs WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if filters.AdminID != nil {
		baseQuery += fmt.Sprintf(" AND admin_id = $%d", argIdx)
		args = append(args, *filters.AdminID)
		argIdx++
	}

	if filters.Action != nil {
		baseQuery += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, *filters.Action)
		argIdx++
	}

	if filters.ResourceType != "" {
		baseQuery += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, filters.ResourceType)
		argIdx++
	}

	if filters.StartDate != nil {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *filters.StartDate)
		argIdx++
	}

	if filters.EndDate != nil {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *filters.EndDate)
		argIdx++
	}

	// Count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Data
	offset := (filters.Page - 1) * filters.PageSize
	dataQuery := "SELECT id, admin_id, admin_email, action, resource_type, resource_id, " +
		"details, ip_address, user_agent, created_at " +
		baseQuery +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filters.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var log domain.AuditLog
		if err := rows.Scan(
			&log.ID,
			&log.AdminID,
			&log.AdminEmail,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&log.Details,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

func (r *PostgresRepository) ListAuditLogsByAdminID(ctx context.Context, adminID uuid.UUID, limit, offset int) ([]domain.AuditLog, int64, error) {
	filters := domain.AuditLogFilters{
		AdminID:  &adminID,
		Page:     1,
		PageSize: limit,
	}

	filters.Page = offset/limit + 1
	return r.ListAuditLogs(ctx, filters)
}

func (r *PostgresRepository) DeleteAuditLogsOlderThan(ctx context.Context, days int) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < $1`
	cutoffDate := time.Now().AddDate(0, 0, -days)

	result, err := r.pool.Exec(ctx, query, cutoffDate)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// ==================== Role Repository (stub implementation) ====================

func (r *PostgresRepository) CreateRole(ctx context.Context, role *domain.Role) error {
	// Stub: roles are typically managed via configuration or migrations
	return fmt.Errorf("role creation not implemented via repository")
}

func (r *PostgresRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	return nil, fmt.Errorf("role retrieval by ID not implemented")
}

func (r *PostgresRepository) GetByName(ctx context.Context, name domain.UserRole) (*domain.Role, error) {
	return &domain.Role{
		ID:          uuid.New(),
		Name:        name,
		Description: fmt.Sprintf("Built-in %s role", name),
		Permissions: nil,
	}, nil
}

func (r *PostgresRepository) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return []domain.Role{
		{Name: domain.RoleSuperAdmin, Description: "Super Administrator with full access"},
		{Name: domain.RoleAdmin, Description: "Administrator with limited access"},
		{Name: domain.RoleModerator, Description: "Moderator with read/update access"},
		{Name: domain.RoleUser, Description: "Regular user with read-only access"},
	}, nil
}

func (r *PostgresRepository) UpdateRoleRecord(ctx context.Context, role *domain.Role) error {
	return fmt.Errorf("role update not implemented via repository")
}

func (r *PostgresRepository) DeleteRoleRecord(ctx context.Context, id uuid.UUID) error {
	return fmt.Errorf("role deletion not implemented via repository")
}
