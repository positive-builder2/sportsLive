package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/positive-builder2/sports/internal/ingestion"
	natspkg "github.com/positive-builder2/sports/pkg/nats"
)

func main() {
	natsURL := envOr("NATS_URL", "nats://localhost:4222")
	matchID := envOr("MATCH_ID", "match-001")
	enableMock := envOr("ENABLE_MOCK_FEED", "false")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if enableMock != "true" {
		log.Printf("ingestion started in passive mode (ENABLE_MOCK_FEED=false). Waiting for manual/admin events...")
		<-ctx.Done()
		return
	}

	nc, err := natspkg.Connect(natsURL)
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer nc.Close()

	svc := ingestion.NewService(ingestion.NewMockAdapter(matchID), nc)

	log.Printf("ingestion auto-seeding active for match=%s nats=%s", matchID, natsURL)
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
