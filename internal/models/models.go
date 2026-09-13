package models

import "time"

type Player struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Team struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Players []Player `json:"players"`
}

type Match struct {
	ID        string    `json:"id"`
	TeamA     Team      `json:"team_a"`
	TeamB     Team      `json:"team_b"`
	Venue     string    `json:"venue"`
	Format    string    `json:"format"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
}

type PlayerStats struct {
	Player    Player `json:"player"`
	Runs      int    `json:"runs"`
	Balls     int    `json:"balls"`
	Fours     int    `json:"fours"`
	Sixes     int    `json:"sixes"`
	Wickets   int    `json:"wickets"`
	OversBowled int  `json:"overs_bowled"`
	RunsConceded int `json:"runs_conceded"`
}

type Partnership struct {
	Runs  int `json:"runs"`
	Balls int `json:"balls"`
}

type InningsState struct {
	ID          string                 `json:"id"`
	Runs        int                    `json:"runs"`
	Wickets     int                    `json:"wickets"`
	Balls       int                    `json:"balls"`
	Overs       int                    `json:"overs"`
	BallInOver  int                    `json:"ball_in_over"`
	Striker     Player                 `json:"striker"`
	NonStriker  Player                 `json:"non_striker"`
	Bowler      Player                 `json:"bowler"`
	Partnership Partnership            `json:"partnership"`
	Batters     map[string]PlayerStats `json:"batters"`
	Bowlers     map[string]PlayerStats `json:"bowlers"`
}

type MatchState struct {
	MatchID     string       `json:"match_id"`
	Status      string       `json:"status"`
	Innings     InningsState `json:"innings"`
	UpdatedAt   time.Time    `json:"updated_at"`
}