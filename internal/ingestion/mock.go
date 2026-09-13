package ingestion

import (
	"context"
	"time"

	"github.com/positive-builder2/sports/internal/events"
)

type MockAdapter struct {
	matchID string
}

func NewMockAdapter(matchID string) *MockAdapter {
	return &MockAdapter{matchID: matchID}
}

func (m *MockAdapter) Connect(ctx context.Context) error {
	return nil
}

func (m *MockAdapter) ReadEvents(ctx context.Context, ch chan<- events.BallDelivered) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	over, ball := 0, 1
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ch <- events.BallDelivered{
				EventID:      time.Now().Format("20060102150405"),
				EventType:    events.TypeBallDelivered,
				MatchID:      m.matchID,
				InningsID:    "inn1",
				Batsman:      events.PlayerRef{ID: "p1", Name: "Kohli"},
				Bowler:       events.PlayerRef{ID: "p2", Name: "Bumrah"},
				Over:         over,
				BallInOver:   ball,
				RunsScored:   1,
				DeliveryType: "pace",
				Timestamp:    time.Now(),
			}
			ball++
			if ball > 6 {
				ball = 1
				over++
			}
		}
	}
}

func (m *MockAdapter) Close() error {
	return nil
}
