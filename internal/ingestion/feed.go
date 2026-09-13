package ingestion

import (
	"context"

	"github.com/positive-builder2/sports/internal/events"
)

type FeedAdapter interface {
	Connect(ctx context.Context) error
	ReadEvents(ctx context.Context, ch chan<- events.BallDelivered) error
	Close() error
}
