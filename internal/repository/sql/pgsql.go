package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	pqUniqueViolationErrCode = "23505" // PostgreSQL unique violation error code. See https://www.postgresql.org/docs/14/errcodes-appendix.html
)

func StartDB(ctx context.Context, dbConf config.DB) (*sql.DB, error) {
	dbCon, err := startDBConnection(dbConf)
	if err != nil {
		slog.Error("failed to initialize DB connection", slog.Any("err", err))
		return nil, fmt.Errorf("failed to initialize DB connection: %w", err)
	}
	slog.Info("DB connection done")
	if err = RunMigrations(dbCon); err != nil {
		slog.Error("failed to run migrations", slog.Any("err", err))
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	slog.Info("DB migration done")
	return dbCon, nil
}

func startDBConnection(conf config.DB) (*sql.DB, error) {
	dsnTmp := "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable"
	dsn := fmt.Sprintf(dsnTmp, conf.Host, conf.User, conf.Password, conf.Name, conf.Port)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func RunMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
