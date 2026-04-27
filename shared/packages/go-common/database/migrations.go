package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/real-ass/shared/go-common/logger"
	"go.uber.org/zap"
)

// MigrationConfig holds configuration for the migration runner.
type MigrationConfig struct {
	// MigrationsPath is the path to the migrations directory.
	// Use "file://path/to/migrations" format.
	MigrationsPath string
	// DatabaseURL is the database connection string for migrations.
	DatabaseURL string
	// TableName is the name of the migration tracking table.
	TableName string
}

// DefaultMigrationConfig returns a MigrationConfig with sensible defaults.
func DefaultMigrationConfig() MigrationConfig {
	return MigrationConfig{
		MigrationsPath: "file://migrations",
		DatabaseURL:    "",
		TableName:      "schema_migrations",
	}
}

// Migrator handles database migrations.
type Migrator struct {
	migrate *migrate.Migrate
	config  MigrationConfig
}

// NewMigrator creates a new Migrator instance.
func NewMigrator(cfg MigrationConfig) (*Migrator, error) {
	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "file://migrations"
	}
	if cfg.TableName == "" {
		cfg.TableName = "schema_migrations"
	}

	return &Migrator{config: cfg}, nil
}

// NewMigratorWithDB creates a Migrator using an existing database connection.
func NewMigratorWithDB(cfg MigrationConfig, db *sql.DB) (*Migrator, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTableName: cfg.TableName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(cfg.MigrationsPath, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return &Migrator{
		migrate: m,
		config:  cfg,
	}, nil
}

// Connect initializes the migrator with the configured database URL.
func (m *Migrator) Connect() error {
	var err error
	m.migrate, err = migrate.New(m.config.MigrationsPath, m.config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	return nil
}

// Close closes the migration instance.
func (m *Migrator) Close() error {
	if m.migrate != nil {
		sourceErr, dbErr := m.migrate.Close()
		if sourceErr != nil {
			return fmt.Errorf("source close error: %w", sourceErr)
		}
		if dbErr != nil {
			return fmt.Errorf("db close error: %w", dbErr)
		}
	}
	return nil
}

// MigrateUp runs all up migrations.
func (m *Migrator) MigrateUp() error {
	if m.migrate == nil {
		if err := m.Connect(); err != nil {
			return err
		}
	}

	logger.Info("running all up migrations")
	err := m.migrate.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}
	if err == migrate.ErrNoChange {
		logger.Info("no migrations to apply")
	} else {
		logger.Info("all up migrations applied successfully")
	}
	return nil
}

// MigrateDown runs all down migrations.
func (m *Migrator) MigrateDown() error {
	if m.migrate == nil {
		if err := m.Connect(); err != nil {
			return err
		}
	}

	logger.Info("running all down migrations")
	err := m.migrate.Down()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run down migrations: %w", err)
	}
	if err == migrate.ErrNoChange {
		logger.Info("no migrations to revert")
	} else {
		logger.Info("all down migrations applied successfully")
	}
	return nil
}

// MigrateTo migrates to a specific version.
func (m *Migrator) MigrateTo(version uint) error {
	if m.migrate == nil {
		if err := m.Connect(); err != nil {
			return err
		}
	}

	logger.Info("migrating to version", zap.Uint("version", version))
	err := m.migrate.Migrate(version)
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}
	if err == migrate.ErrNoChange {
		logger.Info("already at target version", zap.Uint("version", version))
	} else {
		logger.Info("migrated to version", zap.Uint("version", version))
	}
	return nil
}

// MigrateSteps runs a specific number of up/down migrations.
// Positive n runs n up migrations, negative n runs n down migrations.
func (m *Migrator) MigrateSteps(n int) error {
	if m.migrate == nil {
		if err := m.Connect(); err != nil {
			return err
		}
	}

	logger.Info("running migration steps", zap.Int("steps", n))
	err := m.migrate.Steps(n)
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run %d migration steps: %w", n, err)
	}
	return nil
}

// Version returns the current migration version.
func (m *Migrator) Version() (uint, bool, error) {
	if m.migrate == nil {
		if err := m.Connect(); err != nil {
			return 0, false, err
		}
	}

	version, dirty, err := m.migrate.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}
