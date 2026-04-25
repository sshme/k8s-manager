package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"k8s-manager/market/internal"
	"k8s-manager/market/internal/infrastructure/database"
)

func main() {
	var dbConfig database.Config

	flag.StringVar(&dbConfig.Host, "db-host", getEnv("DB_HOST", "localhost"), "Database host")
	flag.IntVar(&dbConfig.Port, "db-port", getEnvInt("DB_PORT", 5432), "Database port")
	flag.StringVar(&dbConfig.User, "db-user", getEnv("DB_USER", "postgres"), "Database user")
	flag.StringVar(&dbConfig.Password, "db-pass", getEnv("DB_PASS", "postgres"), "Database password")
	flag.StringVar(&dbConfig.DBName, "db-name", getEnv("DB_NAME", "k8s_market"), "Database name")
	flag.StringVar(&dbConfig.SSLMode, "db-sslmode", getEnv("DB_SSLMODE", "disable"), "Database SSL mode")

	grpcPort := flag.Int("grpc-port", getEnvInt("GRPC_PORT", 50051), "gRPC server port")
	storagePath := flag.String("storage-path", getEnv("STORAGE_PATH", "./storage"), "Artifact storage path")
	migrationsPath := flag.String("migrations-path", getEnv("MIGRATIONS_PATH", "./migrations"), "SQL migrations directory")

	flag.Parse()

	db, err := database.NewDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Database connection established")

	migrationCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.RunMigrations(migrationCtx, db.DB, *migrationsPath); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations applied")

	app, err := internal.InitializeApp(db.DB, *grpcPort, *storagePath)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
