package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"k8s-manager/market/internal"
	"k8s-manager/market/internal/infrastructure/database"
)

func main() {
	dbHost := flag.String("db-host", getEnv("DB_HOST", "localhost"), "Database host")
	dbPort := flag.Int("db-port", getEnvInt("DB_PORT", 5432), "Database port")
	dbUser := flag.String("db-user", getEnv("DB_USER", "postgres"), "Database user")
	dbPass := flag.String("db-pass", getEnv("DB_PASS", "postgres"), "Database password")
	dbName := flag.String("db-name", getEnv("DB_NAME", "k8s_market"), "Database name")
	dbSSLMode := flag.String("db-sslmode", getEnv("DB_SSLMODE", "disable"), "Database SSL mode")
	grpcPort := flag.Int("grpc-port", getEnvInt("GRPC_PORT", 50051), "gRPC server port")
	storagePath := flag.String("storage-path", getEnv("STORAGE_PATH", "./storage"), "Artifact storage path")

	flag.Parse()

	dbConfig := database.Config{
		Host:     *dbHost,
		Port:     *dbPort,
		User:     *dbUser,
		Password: *dbPass,
		DBName:   *dbName,
		SSLMode:  *dbSSLMode,
	}

	db, err := database.NewDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Database connection established")

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
