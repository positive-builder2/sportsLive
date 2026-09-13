package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/positive-builder2/sports/internal/gateway"
	"github.com/positive-builder2/sports/internal/models"
	natspkg "github.com/positive-builder2/sports/pkg/nats"
	redispkg "github.com/positive-builder2/sports/pkg/redis"
)

func main() {
	natsURL := envOr("NATS_URL", "nats://localhost:4222")
	redisAddr := envOr("REDIS_ADDR", "localhost:6379")
	addr := envOr("GATEWAY_ADDR", ":8080")

	nc, err := natspkg.Connect(natsURL)
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer nc.Close()

	cache := redispkg.Connect(redisAddr)
	defer cache.Close()

	// Seed default match-001 if index is empty
	ctx := context.Background()
	existingMatches, err := cache.GetMatches(ctx)
	if err == nil && len(existingMatches) == 0 {
		defaultMatch := models.Match{
			ID:     "match-001",
			TeamA:  models.Team{ID: "t1", Name: "India"},
			TeamB:  models.Team{ID: "t2", Name: "Australia"},
			Venue:  "MCG, Melbourne",
			Format: "T20",
			Status: "live",
		}
		_ = cache.SaveMatch(ctx, defaultMatch.ID, defaultMatch)
		log.Printf("pre-seeded default match-001 in redis index")
	}

	hub := gateway.NewHub()
	go hub.Run()

	if err := gateway.Subscribe(nc, hub); err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	mux := http.NewServeMux()

	// WebSocket Live Fanout
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		gateway.ServeWS(hub, w, r)
	})

	// Static UI Dashboard Routing
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.ServeFile(w, r, "web/index.html")
			return
		}
		if r.URL.Path == "/admin" || r.URL.Path == "/admin.html" {
			http.ServeFile(w, r, "web/admin.html")
			return
		}
		http.NotFound(w, r)
	})

	// API 1: Register / List Match Metadata
	mux.HandleFunc("/api/matches", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			matches, err := cache.GetMatches(r.Context())
			if err != nil {
				http.Error(w, "redis error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(matches)
			return
		}

		if r.Method == http.MethodPost {
			var match models.Match
			if err := json.NewDecoder(r.Body).Decode(&match); err != nil {
				http.Error(w, "invalid payload", http.StatusBadRequest)
				return
			}
			if match.ID == "" {
				http.Error(w, "match id required", http.StatusBadRequest)
				return
			}
			if err := cache.SaveMatch(r.Context(), match.ID, match); err != nil {
				http.Error(w, "redis save error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"created"}`))
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// API 2: Publish Ball Delivery Event to NATS (Ingestion / Simulator / Admin)
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		if _, err := nc.JS().Publish(r.Context(), natspkg.SubjectNormalized, body); err != nil {
			http.Error(w, "nats publish error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// API 3: Get Single Match Snapshot from Redis Cache
	mux.HandleFunc("/matches/", func(w http.ResponseWriter, r *http.Request) {
		matchID := r.URL.Path[len("/matches/"):]
		if matchID == "" {
			http.Error(w, "match id required", http.StatusBadRequest)
			return
		}
		data, err := cache.GetMatchState(r.Context(), matchID)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("gateway listening on %s nats=%s redis=%s", addr, natsURL, redisAddr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
