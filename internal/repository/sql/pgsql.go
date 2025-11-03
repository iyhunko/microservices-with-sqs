package sql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/iyhunko/microservices-with-sqs/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func StartDB(ctx context.Context, dbConf config.DB) (*sql.DB, error) {
	dbCon, err := startDBConnection(dbConf)
	if err != nil {
		slog.Error("failed to initialize DB connection", slog.Any("err", err))
		return nil, fmt.Errorf("failed to initialize DB connection: %w", err)
	}
	slog.Info("DB connection done")
	if err = Migrate(ctx, dbCon); err != nil {
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
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			region VARCHAR(255) NOT NULL,
			status VARCHAR(255) NOT NULL,
			role VARCHAR(255) NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}
	return nil
}
