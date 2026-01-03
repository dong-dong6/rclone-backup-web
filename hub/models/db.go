package models

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

// InitDB initializes the database connection pool
func InitDB() (*pgxpool.Pool, error) {
	// Build database URL from environment variables
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost" // Default for local development
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "password"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "rclone_backup"
	}

	sslMode := strings.TrimSpace(os.Getenv("DB_SSLMODE"))
	if sslMode == "" {
		sslMode = "require"
	}

	allowWeakPassword := strings.EqualFold(os.Getenv("ALLOW_INSECURE_DB_PASSWORD"), "true")
	if !allowWeakPassword && (dbPassword == "" || dbPassword == "password") {
		return nil, fmt.Errorf("DB_PASSWORD must be set to a non-default value (set ALLOW_INSECURE_DB_PASSWORD=true only for local development)")
	}

	if strings.EqualFold(sslMode, "disable") && !strings.EqualFold(os.Getenv("ALLOW_INSECURE_DB_SSL"), "true") {
		return nil, fmt.Errorf("DB_SSLMODE=disable requires ALLOW_INSECURE_DB_SSL=true (use only for local development)")
	}

	// Check for complete DATABASE_URL first (for compatibility)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Build URL from individual components
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	dbPool = pool
	return pool, nil
}

// GetDB returns the database connection pool
func GetDB() *pgxpool.Pool {
	return dbPool
}
