package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/wf-builder/workflow-builder-backend/internal/config"
	"github.com/wf-builder/workflow-builder-backend/internal/db"
	"github.com/wf-builder/workflow-builder-backend/internal/handlers"
	"github.com/wf-builder/workflow-builder-backend/internal/middleware"
	"github.com/wf-builder/workflow-builder-backend/internal/services"
)

func main() {
	// Load .env from cwd if present (dev convenience)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		// default: ./migrations relative to process cwd
		migrationsDir = "migrations"
		if _, err := os.Stat(migrationsDir); err != nil {
			// try relative to executable's parent (repo root layout)
			if exe, err2 := os.Executable(); err2 == nil {
				candidate := filepath.Join(filepath.Dir(exe), "..", "..", "migrations")
				if _, err3 := os.Stat(candidate); err3 == nil {
					migrationsDir = candidate
				}
			}
		}
	}
	if err := db.RunMigrations(ctx, pool, migrationsDir); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Printf("database connected; migrations checked (%s)", migrationsDir)

	workflows := services.NewWorkflowService(pool, cfg.DefaultUserID)
	nodes := services.NewNodeService(pool, workflows)
	executions := services.NewExecutionService(pool, workflows)
	parser := services.NewParseValuesService(pool)
	webhooks := services.NewWebhookEngine(pool, parser)

	api := &handlers.API{
		Workflows:  workflows,
		Nodes:      nodes,
		Executions: executions,
		Webhooks:   webhooks,
	}

	var handler http.Handler = api.Routes()
	handler = middleware.CORS(cfg.CORS.AllowedOrigins)(handler)
	handler = middleware.Logging(handler)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("workflow-builder backend listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}
