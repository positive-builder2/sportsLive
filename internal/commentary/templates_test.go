package commentary

import (
	"strings"
	"testing"

	"github.com/positive-builder2/sports/internal/events"
)

func TestFour(t *testing.T) {
	got := Line(events.BallDelivered{
		Batsman:    events.PlayerRef{Name: "Kohli"},
		Bowler:     events.PlayerRef{Name: "Bumrah"},
		Over:       18,
		BallInOver: 3,
		RunsScored: 4,
	})
	if !strings.Contains(got, "FOUR") {
		t.Fatalf("got %q", got)
	}
}

func TestWicketBowled(t *testing.T) {
	got := Line(events.BallDelivered{
		Batsman: events.PlayerRef{Name: "Kohli"},
		Bowler:  events.PlayerRef{Name: "Bumrah"},
		Wicket:  &events.Wicket{Type: "bowled"},
	})
	if !strings.Contains(got, "OUT bowled") {
		t.Fatalf("got %q", got)
	}
}

func TestWide(t *testing.T) {
	got := Line(events.BallDelivered{
		Batsman: events.PlayerRef{Name: "Kohli"},
		Bowler:  events.PlayerRef{Name: "Bumrah"},
		Extras:  events.Extras{Wides: 1},
	})
	if !strings.Contains(got, "WIDE") {
		t.Fatalf("got %q", got)
	}
}
