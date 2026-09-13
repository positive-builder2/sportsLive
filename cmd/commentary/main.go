package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/positive-builder2/sports/internal/commentary"
	natspkg "github.com/positive-builder2/sports/pkg/nats"
)

func main() {
	natsURL := envOr("NATS_URL", "nats://localhost:4222")

	nc, err := natspkg.Connect(natsURL)
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer nc.Close()

	svc := commentary.NewService(nc)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("commentary started nats=%s", natsURL)
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
