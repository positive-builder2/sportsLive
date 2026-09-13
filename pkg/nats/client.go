package nats

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	SubjectRaw        = "sports.events.raw"
	SubjectNormalized = "sports.events.normalized"
	SubjectMatchState = "sports.match.%s.state"
	SubjectMatchBall  = "sports.match.%s.ball"
	SubjectCommentary = "sports.commentary.%s"
	SubjectFanout     = "sports.fanout.%s"
)

type Client struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func Connect(url string) (*Client, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	return &Client{nc: nc, js: js}, nil
}

func (c *Client) JS() jetstream.JetStream { return c.js }
func (c *Client) NC() *nats.Conn          { return c.nc }

func (c *Client) Close() {
	c.nc.Close()
}

func MatchBallSubject(matchID string) string {
	return fmt.Sprintf(SubjectMatchBall, matchID)
}

func MatchStateSubject(matchID string) string {
	return fmt.Sprintf(SubjectMatchState, matchID)
}

func CommentarySubject(matchID string) string {
	return fmt.Sprintf(SubjectCommentary, matchID)
}

func FanoutSubject(matchID string) string {
	return fmt.Sprintf(SubjectFanout, matchID)
}
