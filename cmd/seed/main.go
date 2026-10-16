package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"homework/internal/auth"
	"homework/internal/config"
	"homework/internal/repository"
	"homework/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Printf("seed-admin failed: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if cfg.AdminEmail == "" {
		return fmt.Errorf("required environment variable ADMIN_EMAIL is missing")
	}
	if cfg.AdminPassword == "" {
		return fmt.Errorf("required environment variable ADMIN_PASSWORD is missing")
	}

	tokens, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		return fmt.Errorf("JWT_SECRET is invalid: %w", err)
	}

	logLevel := gormlogger.Warn
	if strings.EqualFold(os.Getenv("LOG_SQL"), "true") {
		logLevel = gormlogger.Info
	}
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN()), &gorm.Config{
		Logger:         gormlogger.Default.LogMode(logLevel),
		TranslateError: true,
	})
	if err != nil {
		return fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get PostgreSQL connection pool: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close PostgreSQL connection: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	authService := service.NewAuthService(repository.NewGormUserRepository(db), tokens)
	created, err := authService.EnsureAdmin(ctx, cfg.AdminEmail, cfg.AdminPassword)
	if err != nil {
		return err
	}
	if created {
		log.Printf("seeded admin user %s", cfg.AdminEmail)
		return nil
	}
	log.Printf("admin user %s already exists", cfg.AdminEmail)
	return nil
}
