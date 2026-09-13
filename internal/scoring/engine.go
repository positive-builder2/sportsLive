package scoring

import (
	"time"

	"github.com/positive-builder2/sports/internal/events"
	"github.com/positive-builder2/sports/internal/models"
)

type Engine struct {
	states map[string]*models.MatchState
}

func NewEngine() *Engine {
	return &Engine{states: make(map[string]*models.MatchState)}
}

func (e *Engine) Apply(ball events.BallDelivered) *models.MatchState {
	st, ok := e.states[ball.MatchID]
	if !ok {
		st = newMatchState(ball)
		e.states[ball.MatchID] = st
	}

	// Dynamic player names with fallbacks
	batsmanName := ball.Batsman.Name
	if batsmanName == "" {
		batsmanName = "Batsman 1"
	}
	batsmanID := ball.Batsman.ID
	if batsmanID == "" {
		batsmanID = "p1"
	}

	bowlerName := ball.Bowler.Name
	if bowlerName == "" {
		bowlerName = "Bowler 1"
	}
	bowlerID := ball.Bowler.ID
	if bowlerID == "" {
		bowlerID = "b1"
	}

	legal := isLegalDelivery(ball)
	totalRuns := ball.RunsScored + extraRuns(ball.Extras)

	st.Innings.Runs += totalRuns
	if legal {
		st.Innings.Balls++
		st.Innings.Overs = st.Innings.Balls / 6
		st.Innings.BallInOver = st.Innings.Balls % 6
	}

	st.Innings.Striker = models.Player{ID: batsmanID, Name: batsmanName}
	st.Innings.Bowler = models.Player{ID: bowlerID, Name: bowlerName}

	batter := st.Innings.Batters[batsmanID]
	batter.Player = st.Innings.Striker
	if legal {
		batter.Balls++
	}
	batter.Runs += ball.RunsScored
	if ball.RunsScored == 4 {
		batter.Fours++
	}
	if ball.RunsScored == 6 {
		batter.Sixes++
	}
	st.Innings.Batters[batsmanID] = batter

	bowler := st.Innings.Bowlers[bowlerID]
	bowler.Player = st.Innings.Bowler
	bowler.RunsConceded += totalRuns
	if legal {
		bowler.OversBowled = st.Innings.Overs
	}
	st.Innings.Bowlers[bowlerID] = bowler

	st.Innings.Partnership.Runs += totalRuns
	if legal {
		st.Innings.Partnership.Balls++
	}

	if ball.Wicket != nil {
		st.Innings.Wickets++
		bowler.Wickets++
		st.Innings.Bowlers[bowlerID] = bowler
		st.Innings.Partnership = models.Partnership{}
	}

	if legal && ball.RunsScored%2 == 1 {
		st.Innings.Striker, st.Innings.NonStriker = st.Innings.NonStriker, st.Innings.Striker
	}

	st.UpdatedAt = time.Now()
	return st
}

func (e *Engine) State(matchID string) *models.MatchState {
	return e.states[matchID]
}

func newMatchState(ball events.BallDelivered) *models.MatchState {
	bName := ball.Batsman.Name
	if bName == "" {
		bName = "Batsman 1"
	}
	bwName := ball.Bowler.Name
	if bwName == "" {
		bwName = "Bowler 1"
	}

	return &models.MatchState{
		MatchID: ball.MatchID,
		Status:  "live",
		Innings: models.InningsState{
			ID:      ball.InningsID,
			Striker: models.Player{ID: "p1", Name: bName},
			Bowler:  models.Player{ID: "b1", Name: bwName},
			Batters: make(map[string]models.PlayerStats),
			Bowlers: make(map[string]models.PlayerStats),
		},
		UpdatedAt: time.Now(),
	}
}

func extraRuns(e events.Extras) int {
	return e.Wides + e.NoBalls + e.Byes + e.LegByes
}

func isLegalDelivery(ball events.BallDelivered) bool {
	return ball.Extras.Wides == 0 && ball.Extras.NoBalls == 0
}
