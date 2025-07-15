package database

import (
	"fmt"
	"log"

	"github.com/makseli/medi-pill-check/internal/config"
	"github.com/makseli/medi-pill-check/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.Config) (*gorm.DB, error) {
	var err error
	var dialector gorm.Dialector

	switch cfg.DBType {
	case "postgresql":
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresDB)
		dialector = postgres.Open(dsn)
	case "sqlite":
		fallthrough
	default:
		dialector = sqlite.Open(cfg.SQLitePath)
	}

	DB, err = gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connected successfully")

	// Auto migrate models
	if err := AutoMigrate(DB); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return DB, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{})
}

func GetDB() *gorm.DB {
	return DB
}
