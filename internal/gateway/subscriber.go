package gateway

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
	natspkg "github.com/positive-builder2/sports/pkg/nats"
)

type Envelope struct {
	Type    string          `json:"type"`
	MatchID string          `json:"match_id"`
	Payload json.RawMessage `json:"payload"`
}

func Subscribe(nc *natspkg.Client, hub *Hub) error {
	_, err := nc.NC().Subscribe("sports.match.*.>", func(msg *nats.Msg) {
		matchID, kind := parseSubject(msg.Subject)
		if matchID == "" {
			return
		}
		hub.Broadcast(matchID, Envelope{
			Type:    kind,
			MatchID: matchID,
			Payload: msg.Data,
		})
	})
	if err != nil {
		return err
	}

	_, err = nc.NC().Subscribe("sports.commentary.*", func(msg *nats.Msg) {
		matchID, _ := parseSubject(msg.Subject)
		if matchID == "" {
			return
		}
		hub.Broadcast(matchID, Envelope{
			Type:    "commentary",
			MatchID: matchID,
			Payload: msg.Data,
		})
	})
	if err != nil {
		return err
	}

	log.Println("gateway subscribed to sports.match.*.> and sports.commentary.*")
	return nil
}

func parseSubject(subject string) (matchID, kind string) {
	parts := splitSubject(subject)
	if len(parts) < 3 {
		return "", ""
	}
	if parts[0] == "sports" && parts[1] == "match" && len(parts) >= 4 {
		return parts[2], parts[3]
	}
	if parts[0] == "sports" && parts[1] == "commentary" {
		return parts[2], "commentary"
	}
	return "", ""
}

func splitSubject(s string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return parts
}
