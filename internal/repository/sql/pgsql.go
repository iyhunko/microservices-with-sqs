package sql

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/iyhunko/microservices-with-sqs/internal/config"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StartDB(ctx context.Context, dbConf config.DB) (*gorm.DB, error) {
	dbCon, err := startDBConnection(dbConf)
	if err != nil {
		slog.Error("failed to initialize DB connection", slog.Any("err", err))
		return nil, fmt.Errorf("failed to initialize DB connection: %w", err)
	}
	dbCon = dbCon.WithContext(ctx)
	slog.Info("DB connection done")
	if err = Migrate(dbCon); err != nil {
		slog.Error("failed to run migrations", slog.Any("err", err))
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	slog.Info("DB migration done")
	return dbCon, nil
}

func startDBConnection(conf config.DB) (*gorm.DB, error) {
	dsnTmp := "host=%s user=%s password=%s dbname=%s port=%s"
	dsn := fmt.Sprintf(dsnTmp, conf.Host, conf.User, conf.Password, conf.Name, conf.Port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&model.User{})
}
