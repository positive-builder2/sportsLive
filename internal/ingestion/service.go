package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/positive-builder2/sports/internal/events"
	natspkg "github.com/positive-builder2/sports/pkg/nats"
)

type Service struct {
	adapter FeedAdapter
	nats    *natspkg.Client
}

func NewService(adapter FeedAdapter, nats *natspkg.Client) *Service {
	return &Service{adapter: adapter, nats: nats}
}

func (s *Service) Run(ctx context.Context) error {
	if err := s.adapter.Connect(ctx); err != nil {
		return fmt.Errorf("adapter connect: %w", err)
	}
	defer s.adapter.Close()

	ch := make(chan events.BallDelivered, 64)
	go func() {
		if err := s.adapter.ReadEvents(ctx, ch); err != nil && ctx.Err() == nil {
			log.Printf("read events: %v", err)
		}
		close(ch)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if err := s.publish(ctx, ev); err != nil {
				log.Printf("publish: %v", err)
			}
		}
	}
}

func (s *Service) publish(ctx context.Context, ev events.BallDelivered) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if _, err := s.nats.JS().Publish(ctx, natspkg.SubjectNormalized, data); err != nil {
		return fmt.Errorf("nats publish: %w", err)
	}
	log.Printf("published ball match=%s over=%d.%d runs=%d", ev.MatchID, ev.Over, ev.BallInOver, ev.RunsScored)
	return nil
}
