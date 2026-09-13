package events

import "time"

const (
	TypeMatchStarted    = "match.started"
	TypeMatchEnded      = "match.ended"
	TypeInningsStarted  = "innings.started"
	TypeInningsEnded    = "innings.ended"
	TypeOverCompleted   = "over.completed"
	TypeBallDelivered   = "ball.delivered"
)

type PlayerRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Extras struct {
	Wides    int `json:"wides"`
	NoBalls  int `json:"no_balls"`
	Byes     int `json:"byes"`
	LegByes  int `json:"leg_byes"`
}

type Wicket struct {
	Type       string    `json:"type"`
	PlayerOut  PlayerRef `json:"player_out"`
	Fielder    *PlayerRef `json:"fielder,omitempty"`
}

type BallDelivered struct {
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	MatchID     string    `json:"match_id"`
	InningsID   string    `json:"innings_id"`
	Batsman     PlayerRef `json:"batsman"`
	Bowler      PlayerRef `json:"bowler"`
	Over        int       `json:"over"`
	BallInOver  int       `json:"ball_in_over"`
	RunsScored  int       `json:"runs_scored"`
	Extras      Extras    `json:"extras"`
	Wicket      *Wicket   `json:"wicket,omitempty"`
	DeliveryType string   `json:"delivery_type"`
	Timestamp   time.Time `json:"timestamp"`
}

type OverCompleted struct {
	EventType     string    `json:"event_type"`
	MatchID       string    `json:"match_id"`
	InningsID     string    `json:"innings_id"`
	OverNumber    int       `json:"over_number"`
	RunsInOver    int       `json:"runs_in_over"`
	WicketsInOver int       `json:"wickets_in_over"`
	ScoreAtOver   Score     `json:"score_at_over"`
	Timestamp     time.Time `json:"timestamp"`
}

type Score struct {
	Runs    int `json:"runs"`
	Wickets int `json:"wickets"`
	Balls   int `json:"balls"`
}
