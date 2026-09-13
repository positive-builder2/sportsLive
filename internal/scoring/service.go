package scoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/positive-builder2/sports/internal/events"
	"github.com/positive-builder2/sports/pkg/db"
	natspkg "github.com/positive-builder2/sports/pkg/nats"
	redispkg "github.com/positive-builder2/sports/pkg/redis"
)

type Service struct {
	engine *Engine
	nats   *natspkg.Client
	cache  *redispkg.Client
	store  *db.Client
}

func NewService(engine *Engine, nats *natspkg.Client, cache *redispkg.Client, store *db.Client) *Service {
	return &Service{engine: engine, nats: nats, cache: cache, store: store}
}

func (s *Service) Run(ctx context.Context) error {
	js := s.nats.JS()

	// Ensure streams exist
	var stream jetstream.Stream
	var err error

	// Ensure stream exists
	ctxStream, cancelStream := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStream()
	_, err = js.CreateOrUpdateStream(ctxStream, jetstream.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"sports.events.>"},
	})
	if err != nil {
		log.Printf("scoring: failed to create stream: %v", err)
	}
	for i := 0; i < 10; i++ {
		stream, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
			Name:     "EVENTS",
			Subjects: []string{"sports.events.*"},
		})
		if err == nil {
			break
		}
		log.Printf("retrying stream creation: %v", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("stream: %w", err)
	}

	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:              "MATCH_STATE",
		Subjects:          []string{"sports.match.*.state"},
		Retention:         jetstream.WorkQueuePolicy,
		Discard:           jetstream.DiscardOld,
		MaxMsgsPerSubject: 1,
	}); err != nil {
		return fmt.Errorf("state stream: %w", err)
	}

	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "MATCH_BALLS",
		Subjects: []string{"sports.match.*.ball"},
	}); err != nil {
		return fmt.Errorf("balls stream: %w", err)
	}

	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "scoring",
		FilterSubject: natspkg.SubjectNormalized,
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

	log.Println("scoring engine consuming sports.events.normalized")

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

	state := s.engine.Apply(ball)

	// Assign computed over and ball-in-over back to the ball event before publishing
	ball.Over = state.Innings.Overs
	ball.BallInOver = state.Innings.BallInOver

	ballData, err := json.Marshal(ball)
	if err != nil {
		return fmt.Errorf("marshal ball: %w", err)
	}
	if _, err := s.nats.JS().Publish(ctx, natspkg.MatchBallSubject(ball.MatchID), ballData); err != nil {
		return fmt.Errorf("publish ball: %w", err)
	}

	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if _, err := s.nats.JS().Publish(ctx, natspkg.MatchStateSubject(ball.MatchID), stateData); err != nil {
		return fmt.Errorf("publish state: %w", err)
	}

	if s.cache != nil {
		if err := s.cache.SetMatchState(ctx, ball.MatchID, state); err != nil {
			log.Printf("redis: %v", err)
		}
	}
	if s.store != nil {
		if err := s.store.InsertBall(ctx, ball); err != nil {
			log.Printf("postgres: %v", err)
		}
	}

	log.Printf("scored match=%s %d/%d over=%d.%d",
		state.MatchID, state.Innings.Runs, state.Innings.Wickets,
		state.Innings.Overs, state.Innings.BallInOver)
	return nil
}
