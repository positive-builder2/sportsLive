package commentary

import (
	"fmt"

	"github.com/positive-builder2/sports/internal/events"
)

func Line(ball events.BallDelivered) string {
	over := fmt.Sprintf("%d.%d", ball.Over, ball.BallInOver)
	batter := ball.Batsman.Name
	bowler := ball.Bowler.Name

	if ball.Wicket != nil {
		return wicketLine(over, batter, bowler, ball)
	}

	extras := extraLine(over, batter, bowler, ball)
	if extras != "" {
		return extras
	}

	switch ball.RunsScored {
	case 0:
		return fmt.Sprintf("%s %s to %s, no run", over, bowler, batter)
	case 1:
		return fmt.Sprintf("%s %s to %s, 1 run", over, bowler, batter)
	case 2:
		return fmt.Sprintf("%s %s to %s, 2 runs", over, bowler, batter)
	case 3:
		return fmt.Sprintf("%s %s to %s, 3 runs", over, bowler, batter)
	case 4:
		return fmt.Sprintf("%s %s to %s, FOUR", over, bowler, batter)
	case 6:
		return fmt.Sprintf("%s %s to %s, SIX", over, bowler, batter)
	default:
		return fmt.Sprintf("%s %s to %s, %d runs", over, bowler, batter, ball.RunsScored)
	}
}

func wicketLine(over, batter, bowler string, ball events.BallDelivered) string {
	w := ball.Wicket
	switch w.Type {
	case "bowled":
		return fmt.Sprintf("%s %s to %s, OUT bowled", over, bowler, batter)
	case "caught":
		if w.Fielder != nil {
			return fmt.Sprintf("%s %s to %s, OUT caught by %s", over, bowler, batter, w.Fielder.Name)
		}
		return fmt.Sprintf("%s %s to %s, OUT caught", over, bowler, batter)
	case "lbw":
		return fmt.Sprintf("%s %s to %s, OUT LBW", over, bowler, batter)
	case "run_out":
		if w.Fielder != nil {
			return fmt.Sprintf("%s %s to %s, OUT run out (%s)", over, bowler, batter, w.Fielder.Name)
		}
		return fmt.Sprintf("%s %s to %s, OUT run out", over, bowler, batter)
	case "stumped":
		return fmt.Sprintf("%s %s to %s, OUT stumped", over, bowler, batter)
	default:
		return fmt.Sprintf("%s %s to %s, OUT %s", over, bowler, batter, w.Type)
	}
}

func extraLine(over, batter, bowler string, ball events.BallDelivered) string {
	e := ball.Extras
	switch {
	case e.Wides > 0:
		return fmt.Sprintf("%s %s to %s, WIDE, %d extra", over, bowler, batter, e.Wides)
	case e.NoBalls > 0:
		return fmt.Sprintf("%s %s to %s, NO BALL, %d extra", over, bowler, batter, e.NoBalls)
	case e.Byes > 0:
		return fmt.Sprintf("%s %s to %s, %d bye", over, bowler, batter, e.Byes)
	case e.LegByes > 0:
		return fmt.Sprintf("%s %s to %s, %d leg bye", over, bowler, batter, e.LegByes)
	default:
		return ""
	}
}
