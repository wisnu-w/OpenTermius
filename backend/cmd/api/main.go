package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	httpAdapter "github.com/opentermius/opentermius/internal/adapters/handler/http"
	"github.com/opentermius/opentermius/internal/adapters/repository/postgres"
	"github.com/opentermius/opentermius/internal/usecase"
)

func main() {
	loadEnvFile(".env", "../.env")

	port := getEnv("SERVER_PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/opentermius?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-opentermius-key-32-chars-long!")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("Starting OpenTermius Backend API...")

	// 1. Initialize PostgreSQL Connection Pool
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Invalid DATABASE_URL: %v", err)
	}
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 5 * time.Minute

	// Connect with bounded timeout
	connectCtx, cancelConnect := context.WithTimeout(ctx, 5*time.Second)
	defer cancelConnect()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		log.Printf("Warning: Failed to connect to database immediately (%v). Running in bootstrap mode.", err)
	} else {
		defer pool.Close()
		log.Println("Database connection pool initialized.")
	}

	// 2. Initialize Repositories (Outbound Adapters)
	userRepo := postgres.NewUserRepository(pool)
	vaultRepo := postgres.NewVaultRepository(pool)

	// 3. Initialize UseCases (Application Business Rules)
	authUC := usecase.NewAuthUseCase(userRepo, jwtSecret, 72*time.Hour)
	vaultUC := usecase.NewVaultUseCase(vaultRepo, userRepo)

	// 3b. Ensure Default Gateway Admin Account Exists
	if pool != nil {
		adminEmail := getEnv("DEFAULT_ADMIN_EMAIL", "admin@opentermius.local")
		adminPass := getEnv("DEFAULT_ADMIN_PASSWORD", "OpenTermius2026!")
		adminName := getEnv("DEFAULT_ADMIN_NAME", "Gateway Admin")
		if _, _, err := authUC.Register(ctx, adminEmail, adminPass, adminName); err == nil {
			log.Printf("Seeded default Gateway Admin user: %s", adminEmail)
		}
	}

	// 4. Initialize Handlers
	authHandler := httpAdapter.NewAuthHandler(authUC)
	vaultHandler := httpAdapter.NewVaultHandler(vaultUC)

	// 5. Initialize Router
	router := httpAdapter.NewRouter(authUC, authHandler, vaultHandler)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("OpenTermius Gateway HTTP server listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failure: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server gracefully...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped.")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func loadEnvFile(paths ...string) {
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				if os.Getenv(k) == "" {
					_ = os.Setenv(k, v)
				}
			}
		}
	}
}
