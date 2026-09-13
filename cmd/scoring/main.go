package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/positive-builder2/sports/internal/scoring"
	"github.com/positive-builder2/sports/pkg/db"
	natspkg "github.com/positive-builder2/sports/pkg/nats"
	redispkg "github.com/positive-builder2/sports/pkg/redis"
)

func main() {
	natsURL := envOr("NATS_URL", "nats://localhost:4222")
	redisAddr := envOr("REDIS_ADDR", "localhost:6379")
	pgURL := envOr("DATABASE_URL", "postgres://sports:sports@localhost:5432/sports?sslmode=disable")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	nc, err := natspkg.Connect(natsURL)
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer nc.Close()

	cache := redispkg.Connect(redisAddr)
	defer cache.Close()
	if err := cache.Ping(ctx); err != nil {
		log.Fatalf("redis: %v", err)
	}

	store, err := db.Connect(ctx, pgURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer store.Close()

	svc := scoring.NewService(scoring.NewEngine(), nc, cache, store)

	log.Printf("scoring started nats=%s redis=%s", natsURL, redisAddr)
	if err := svc.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("run: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
