package commentary

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/positive-builder2/sports/internal/events"
	natspkg "github.com/positive-builder2/sports/pkg/nats"
	"github.com/nats-io/nats.go/jetstream"
)

type LineEvent struct {
	MatchID   string    `json:"match_id"`
	InningsID string    `json:"innings_id"`
	Over      int       `json:"over"`
	Ball      int       `json:"ball"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type Service struct {
	nats *natspkg.Client
}

func NewService(nats *natspkg.Client) *Service {
	return &Service{nats: nats}
}

func (s *Service) Run(ctx context.Context) error {
	js := s.nats.JS()

	// Ensure stream exists for match balls
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "MATCH_BALLS",
		Subjects: []string{"sports.match.*.ball"},
	})
	if err != nil {
		return fmt.Errorf("stream: %w", err)
	}

	// Create JetStream stream for commentary as well so JetStream publishes succeed
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "COMMENTARY",
		Subjects: []string{"sports.commentary.*"},
	}); err != nil {
		log.Printf("commentary stream setup warning: %v", err)
	}

	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "commentary",
		FilterSubject: "sports.match.*.ball",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("consumer: %w", err)
	}

	iter, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("messages: %w", err)
	}
	defer iter.Stop()

	log.Println("commentary consuming sports.match.*.ball")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := iter.Next()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("next: %v", err)
				continue
			}
			if err := s.handle(ctx, msg); err != nil {
				log.Printf("handle: %v", err)
			}
			_ = msg.Ack()
		}
	}
}

func (s *Service) handle(ctx context.Context, msg jetstream.Msg) error {
	var ball events.BallDelivered
	if err := json.Unmarshal(msg.Data(), &ball); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	line := LineEvent{
		MatchID:   ball.MatchID,
		InningsID: ball.InningsID,
		Over:      ball.Over,
		Ball:      ball.BallInOver,
		Text:      Line(ball),
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(line)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	// Publish via core NATS for real-time WebSocket fanout
	if err := s.nats.NC().Publish(natspkg.CommentarySubject(ball.MatchID), data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	log.Printf("commentary match=%s %s", ball.MatchID, line.Text)
	return nil
}
