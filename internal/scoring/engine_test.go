package scoring

import (
	"testing"
	"time"

	"github.com/positive-builder2/sports/internal/events"
)

func TestApplyLegalBall(t *testing.T) {
	e := NewEngine()
	st := e.Apply(ball(1, 0, 0))
	if st.Innings.Runs != 1 {
		t.Fatalf("runs=%d want 1", st.Innings.Runs)
	}
	if st.Innings.Balls != 1 {
		t.Fatalf("balls=%d want 1", st.Innings.Balls)
	}
}

func TestApplyWideDoesNotCountBall(t *testing.T) {
	e := NewEngine()
	st := e.Apply(events.BallDelivered{
		MatchID:    "m1",
		InningsID:  "inn1",
		Batsman:    events.PlayerRef{ID: "p1", Name: "Kohli"},
		Bowler:     events.PlayerRef{ID: "p2", Name: "Bumrah"},
		RunsScored: 0,
		Extras:     events.Extras{Wides: 1},
		Timestamp:  time.Now(),
	})
	if st.Innings.Runs != 1 {
		t.Fatalf("runs=%d want 1", st.Innings.Runs)
	}
	if st.Innings.Balls != 0 {
		t.Fatalf("balls=%d want 0", st.Innings.Balls)
	}
}

func TestWicketResetsPartnership(t *testing.T) {
	e := NewEngine()
	e.Apply(ball(4, 0, 0))
	st := e.Apply(events.BallDelivered{
		MatchID:   "m1",
		InningsID: "inn1",
		Batsman:   events.PlayerRef{ID: "p1", Name: "Kohli"},
		Bowler:    events.PlayerRef{ID: "p2", Name: "Bumrah"},
		Wicket:    &events.Wicket{Type: "bowled", PlayerOut: events.PlayerRef{ID: "p1", Name: "Kohli"}},
		Timestamp: time.Now(),
	})
	if st.Innings.Wickets != 1 {
		t.Fatalf("wickets=%d want 1", st.Innings.Wickets)
	}
	if st.Innings.Partnership.Runs != 0 {
		t.Fatalf("partnership=%d want 0", st.Innings.Partnership.Runs)
	}
}

func ball(runs, over, n int) events.BallDelivered {
	return events.BallDelivered{
		MatchID:    "m1",
		InningsID:  "inn1",
		Batsman:    events.PlayerRef{ID: "p1", Name: "Kohli"},
		Bowler:     events.PlayerRef{ID: "p2", Name: "Bumrah"},
		Over:       over,
		BallInOver: n,
		RunsScored: runs,
		Timestamp:  time.Now(),
	}
}
