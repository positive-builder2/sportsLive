package db

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/positive-builder2/sports/internal/events"
)

//go:embed schema.sql
var schemaFS embed.FS

type Client struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, url string) (*Client, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("pgx: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	c := &Client{pool: pool}
	if err := c.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Close() {
	c.pool.Close()
}

func (c *Client) migrate(ctx context.Context) error {
	sql, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if _, err := c.pool.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

func (c *Client) EnsureMatch(ctx context.Context, matchID string) error {
	_, err := c.pool.Exec(ctx,
		`INSERT INTO matches (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`,
		matchID,
	)
	return err
}

func (c *Client) InsertBall(ctx context.Context, ball events.BallDelivered) error {
	if err := c.EnsureMatch(ctx, ball.MatchID); err != nil {
		return err
	}

	var wicketType, playerOut *string
	if ball.Wicket != nil {
		wicketType = &ball.Wicket.Type
		playerOut = &ball.Wicket.PlayerOut.ID
	}

	_, err := c.pool.Exec(ctx, `
		INSERT INTO balls (
			event_id, match_id, innings_id, over_number, ball_in_over,
			batsman_id, batsman_name, bowler_id, bowler_name,
			runs_scored, extras_wides, extras_noballs, extras_byes, extras_legbyes,
			wicket_type, player_out_id, delivery_type, delivered_at
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,
			$10,$11,$12,$13,$14,
			$15,$16,$17,$18
		)
		ON CONFLICT (event_id) DO NOTHING
	`,
		ball.EventID, ball.MatchID, ball.InningsID, ball.Over, ball.BallInOver,
		ball.Batsman.ID, ball.Batsman.Name, ball.Bowler.ID, ball.Bowler.Name,
		ball.RunsScored, ball.Extras.Wides, ball.Extras.NoBalls, ball.Extras.Byes, ball.Extras.LegByes,
		wicketType, playerOut, ball.DeliveryType, ball.Timestamp,
	)
	return err
}
