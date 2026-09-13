1. High-Level Architecture Overview
This repository implements a multi-service, event-driven, real-time sports scoring, commentary generation, and live fanout system built for cricket matches. The architecture follows a microservice model where autonomous Go background daemons communicate asynchronously over NATS JetStream and Core NATS, with persistent storage split between Redis (for low-latency caching of current match state and match indices) and PostgreSQL (for append-only, idempotent historical ball-by-ball event persistence).


<img width="1223" height="1286" alt="image" src="https://github.com/user-attachments/assets/9107fe03-c27b-4ace-b48e-515089cc60e9" />

2. End-to-End Data Flow
->Ingestion Layer:

Ball delivery events enter the system via the ingestion service (cmd/ingestion/main.go) using a ticker-driven MockAdapter or directly via HTTP POST requests to /api/events on the Gateway.
Events are published as JSON payloads to the NATS JetStream subject sports.events.normalized inside the EVENTS stream.

->Scoring Engine Processing:
The scoring service listens on sports.events.normalized via a durable JetStream consumer named scoring.
The raw BallDelivered event is passed to an in-memory state engine (Engine.Apply()).
The engine computes cumulative team score (runs, wickets, overs, balls in over), updates individual striker and bowler statistics (runs scored, boundaries, overs bowled, runs conceded, wickets), handles strike rotation (swapping striker/non-striker on odd runs at the end of legal deliveries), and updates partnership statistics.
The updated computed MatchState and enriched BallDelivered events are fanout published to NATS subjects sports.match.<ID>.state and sports.match.<ID>.ball.
Simultaneously, scoring caches the latest MatchState snapshot in Redis key match:<ID>:state and executes an idempotent SQL INSERT INTO balls query in PostgreSQL.

->Commentary Generation:

The commentary service listens on sports.match.<ID>.ball via a durable consumer named commentary.
It formats the ball data using Line(ball) into natural language text (e.g., "0.1 Bumrah to Kohli, FOUR" or "0.4 Bumrah to Kohli, OUT caught").
It publishes the formatted string wrapped inside a LineEvent payload to Core NATS on sports.commentary.<ID>.


->Gateway & Real-Time Broadcast:
The gateway service maintains background Core NATS wild-card subscriptions to sports.match.*.> and sports.commentary.*.
Incoming NATS messages are encapsulated in a standard { type, match_id, payload } WebSocket envelope.
The Gateway's concurrent Hub broadcasts the message down active WebSocket channels to client connections subscribed to that specific match_id room (or the "all" global room).
Web dashboards render live scoreboards, wagon-wheel indicators, and ball-by-ball commentary feeds in real time.

3. Core Technologies & Dependencies
Language: Go 1.26.5 (net/http, context, sync, embed, time, os/signal).
Message Broker / Streaming: NATS 2.11 with JetStream enabled (github.com/nats-io/nats.go, github.com/nats-io/nats.go/jetstream).
In-Memory Store / Cache: Redis 7 (github.com/redis/go-redis/v9).
Relational Database: PostgreSQL 16 (github.com/jackc/pgx/v5 with connection pooling via pgxpool).
WebSockets: Gorilla WebSocket (github.com/gorilla/websocket).
Containerization & Orchestration: Docker (Dockerfile multi-stage targets) and Docker Compose (docker-compose.yml).
Development Automation: GNU Makefile and POSIX shell script (scripts/dev.sh).


4. Architectural Design Patterns & Principles
->Event-Driven Architecture (EDA): Decoupled producer/consumer microservices connected exclusively via topic subjects over NATS JetStream.
->Domain-Driven In-Memory State Machine: The scoring engine (internal/scoring/engine.go) is a pure Go state machine with zero direct IO side effects, simplifying unit testing (engine_test.go).
->CQRS / Event Sourcing Hybrids: State transitions are triggered strictly by appending raw BallDelivered event records. Read models are served out of Redis snapshots or WebSocket feeds, while write operations append to Postgres.
->Concurrent WebSocket Room Hub Pattern: Traditional Go channel-based hub (register, unregister, broadcast) coupled with decoupled readPump and writePump goroutines per WebSocket client connection.
->Adapter Pattern for Ingestion: FeedAdapter interface decouples mock simulators from live data providers (e.g. external provider REST/WebSocket feeds).
->Embedded Database Migrations: Uses Go //go:embed schema.sql to guarantee schema migrations automatically run upon service startup without external migration tool binaries.


5. Main Services & Infrastructure Clients
sports/
├── cmd/                       # Service Entrypoints
│   ├── ingestion/main.go      # Data feed simulator / ingest runner
│   ├── scoring/main.go        # State machine processing engine
│   ├── commentary/main.go     # Text template commentary engine
│   └── gateway/main.go        # HTTP API + WS fanout server
├── internal/                  # Internal Domain Logic & Services
│   ├── commentary/            # Template generation and commentary service
│   ├── events/                # Event schema definitions (NATS payload contracts)
│   ├── gateway/               # WebSocket Hub, connection pumps, NATS router
│   ├── ingestion/             # Ingestion interface, mock feed generator
│   ├── models/                # Core domain state models (Match, Innings, Stats)
│   └── scoring/               # Scoring state engine & NATS handler
├── pkg/                       # Reusable Infrastructure Packages
│   ├── db/                    # PostgreSQL pgx pool client & SQL schema
│   ├── nats/                  # NATS & JetStream client wrapper & subject constants
│   └── redis/                 # Redis Go-Redis client wrapper
├── web/                       # Static Web Dashboard Frontend UIs
│   ├── admin.html             # Administrative match manager & ball simulator
│   ├── index.html             # Main match overview dashboard
│   └── view.html              # Dedicated live score view page
├── scripts/dev.sh             # Local multi-process development script
├── docker-compose.yml         # Container topology definition
├── Dockerfile                 # Multi-target Docker binary builder
└── Makefile                   # Command shortcuts for run, build, test, dev

6. Detailed File-by-File Guide
Entry Points (cmd/)
cmd/ingestion/main.go
Purpose: Entry point for the ingestion service.
Key Functions / Types: main(), envOr().
Behavior: Checks environment variable ENABLE_MOCK_FEED. If "true", initializes ingestion.NewMockAdapter(matchID) and runs ingestion.Service. If "false" (default in passive mode), it runs passively, allowing administrative API pushes to populate data.
cmd/scoring/main.go
Purpose: Entry point for the scoring engine daemon.
Key Functions / Types: main(), envOr().
Behavior: Connects to NATS, Redis, and PostgreSQL. Instantiates scoring.NewEngine() and starts scoring.Service.Run(). Operates as the central state calculator of the platform.
cmd/commentary/main.go
Purpose: Entry point for the real-time text commentary service.
Key Functions / Types: main(), envOr().
Behavior: Connects to NATS and executes commentary.Service.Run(), processing match ball streams into formatted commentary text events.
cmd/gateway/main.go
Purpose: API Gateway, WebSocket Hub, and static HTTP server entry point.
Key Functions / Types: main(), HTTP Handlers (/ws, /, /api/matches, /api/events, /matches/*, /health).
Behavior:
Seeds match-001 metadata into Redis if index is empty.
Starts concurrent gateway.Hub.
Subscribes to NATS streams via gateway.Subscribe().
Serves static Web UI files (index.html, admin.html).
Implements REST endpoints to fetch match metadata, fetch match snapshot, or publish manual ball events directly to NATS.
Internal Domain Logic (internal/)
internal/models/models.go
Purpose: Data structures for cricket match domain entities and state snapshots.
Key Structures:
Player, Team, Match: Match metadata entities.
PlayerStats: Individual batting (runs, balls, 4s, 6s) and bowling stats (wickets, overs, runs conceded).
Partnership: Current wicket partnership tracking.
InningsState: Real-time state of an innings (total runs, wickets, overs, ball in over, current striker, non-striker, bowler, batters/bowlers maps).
MatchState: High-level wrapper tracking overall match status and active innings state.
internal/events/events.go
Purpose: Event payload contracts transmitted across NATS subjects.
Key Constants / Structures:
Event type string constants (TypeMatchStarted, TypeBallDelivered, TypeOverCompleted, etc.).
PlayerRef: Compact player reference (ID, Name).
Extras: Granular breakdown of illegal/extra deliveries (Wides, NoBalls, Byes, LegByes).
Wicket: Wicket metadata (Type, PlayerOut, Fielder).
BallDelivered: Complete payload representing a single delivery event.
OverCompleted: Event emitted upon completion of a 6-ball over.
internal/ingestion/feed.go
Purpose: Abstraction interface for event feeds.
Key Interface: FeedAdapter (defines Connect(), ReadEvents(), Close()).
internal/ingestion/mock.go
Purpose: Mock implementation of FeedAdapter for automated match simulation.
Key Functions / Types: MockAdapter, NewMockAdapter(), ReadEvents().
Behavior: Runs a 3-second ticker loop generating synthetic BallDelivered events (incrementing balls and overs sequentially) and pushes them to a Go channel.
internal/ingestion/service.go
Purpose: Orchestrates reading from a FeedAdapter and publishing to NATS JetStream.
Key Functions / Types: Service, NewService(), Run(), publish().
Behavior: Spawns a reader goroutine that consumes events from FeedAdapter and publishes them to NATS subject sports.events.normalized.
internal/scoring/engine.go
Purpose: Core in-memory state engine for cricket rule calculations.
Key Functions / Types: Engine, NewEngine(), Apply(ball), isLegalDelivery(), extraRuns().
Behavior:
Computes total runs, extra runs, and legal ball count increments.
Updates striker and bowler stats in st.Innings.Batters and st.Innings.Bowlers.
Increments wickets and resets partnership on dismissals.
Rotates strike between striker and non-striker on odd runs (1, 3, 5) off legal balls.
internal/scoring/engine_test.go
Purpose: Unit test suite for scoring.Engine.
Key Tests: TestEngine_Apply() verifies cumulative score calculation, ball progression, and boundary tracking across consecutive deliveries.
internal/scoring/service.go
Purpose: Service coordinator connecting NATS JetStream, Redis, PostgreSQL, and Engine.
Key Functions / Types: Service, NewService(), Run(), handle().
Behavior:
Creates JetStream streams EVENTS, MATCH_STATE, MATCH_BALLS.
Creates durable consumer scoring on sports.events.normalized.
On each message: runs engine.Apply(), publishes updated ball and state events to NATS JetStream, writes state snapshot to Redis, and saves raw ball event to Postgres.
internal/commentary/templates.go
Purpose: Natural language commentary line generator.
Key Functions: Line(), wicketLine(), extraLine().
Behavior: Matches ball properties (wickets, wide/no-ball extras, runs scored: 0, 1, 2, 3, FOUR, SIX) and produces human-readable strings like "0.1 Bumrah to Kohli, FOUR".
internal/commentary/templates_test.go
Purpose: Unit tests for commentary template generation logic.
Key Tests: TestLine() checks string outputs for regular deliveries, fours, sixes, and wicket events.
internal/commentary/service.go
Purpose: JetStream consumer for match balls that generates and broadcasts text commentary.
Key Functions / Types: Service, NewService(), Run(), handle().
Behavior: Subscribes to sports.match.*.ball via JetStream consumer commentary, generates commentary string via Line(ball), and publishes LineEvent to Core NATS topic sports.commentary.<match_id>.
internal/gateway/hub.go
Purpose: Thread-safe WebSocket client manager and room broadcaster.
Key Functions / Types: Client, Hub, NewHub(), Run(), Broadcast(), Register(), Unregister().
Behavior: Maintains a nested map of rooms map[matchID]map[*Client]struct{} protected by sync.RWMutex. Handles concurrent client registrations, unregistrations, and channel message pushes to specific match rooms and global "all" rooms.
internal/gateway/ws.go
Purpose: Upgrades HTTP connections to WebSockets and handles network IO.
Key Functions: ServeWS(), readPump(), writePump().
Behavior: Sets up Gorilla WebSocket upgrader, registers clients to Hub based on match_id query param, handles ping/pong keepalives, and flushes messages to network sockets.
internal/gateway/subscriber.go
Purpose: Bridges NATS pub/sub messages into the gateway Hub.
Key Functions: Subscribe(), parseSubject(), splitSubject().
Behavior: Sets up Core NATS subscriptions for sports.match.*.> and sports.commentary.*, parses match IDs from subjects, wraps payloads in an Envelope, and routes them to hub.Broadcast().
Infrastructure Packages (pkg/)
pkg/nats/client.go
Purpose: NATS and JetStream client initialization wrapper and subject routing helpers.
Key Constants: Subject definitions (SubjectRaw, SubjectNormalized, SubjectMatchState, SubjectMatchBall, SubjectCommentary, SubjectFanout).
Key Functions: Connect(), MatchBallSubject(), MatchStateSubject(), CommentarySubject(), FanoutSubject().
pkg/redis/client.go
Purpose: Redis client wrapper for match state caching and indexing.
Key Functions: Connect(), SetMatchState(), GetMatchState(), SaveMatch(), GetMatches().
Behavior: Uses Redis string key match:<id>:state with a 24-hour TTL and a Redis Set matches:index to track active match IDs.
pkg/db/schema.sql
Purpose: Relational schema DDL for PostgreSQL database.
Tables:
matches (id, status, created_at).
balls (id, event_id UNIQUE, match_id, innings_id, over_number, ball_in_over, batsman_id, batsman_name, bowler_id, bowler_name, runs_scored, extras_*, wicket_type, player_out_id, delivery_type, delivered_at).
Indexes: balls_match_idx on (match_id, delivered_at).
pkg/db/client.go
Purpose: PostgreSQL connection pool client and query executor.
Key Functions: Connect(), migrate(), EnsureMatch(), InsertBall().
Behavior: Uses //go:embed schema.sql to execute DDL automatically on connection. Performs idempotent inserts into balls using ON CONFLICT (event_id) DO NOTHING.
Web Frontend (web/)
web/index.html
Purpose: Primary real-time match dashboard.
Features: Live score banner, overs progress, live commentary timeline, scorecard table, wagon wheel visualizer, and WebSocket connection state indicator.
web/admin.html
Purpose: Operator administration panel and simulator console.
Features: Form interface to manually submit custom ball deliveries (runs, extra types, wicket types, player names) to /api/events, create new matches, and toggle auto-simulation feeds.
web/view.html
Purpose: Minimal clean scoreboard UI meant for embedding or standalone viewing.
Operations & Configuration
docker-compose.yml
Services: nats (alpine with JetStream -js enabled), redis (7-alpine), postgres (16-alpine with healthcheck), scoring, commentary, gateway, ingestion.
Networks & Ports: Maps NATS 4222/8222, Redis 6379, Postgres 5432, Gateway 8080.
Dockerfile
Structure: Multi-stage Go build (golang:1.26-alpine build stage producing small static alpine binaries targeting specific binary build args SERVICE).
Makefile
Targets: nats, ingestion, scoring, commentary, gateway, dev, up, down, logs, test, build.
scripts/dev.sh
Purpose: Launches infrastructure dependencies in Docker (nats, redis, postgres) and starts all 4 Go services in parallel in a single terminal session with trap handlers to gracefully kill background processes on SIGINT/Ctrl+C.



7. Setup & Execution Guide
Prerequisite Infrastructure
Ensure Docker and Go 1.26+ are installed on your host system.

Running with Docker Compose (Full Stack)
# Build images and launch all services in detached mode
make up

# View consolidated live service logs
make logs

# Tear down container stack
make down


Local Hybrid Development (Containers + Local Go Services)
# Start NATS, Redis, and Postgres containers in background, then run Go services locally
make dev


Access Main Dashboard: http://localhost:8080/
Access Admin / Simulator: http://localhost:8080/admin
Health Check: http://localhost:8080/health

Running Tests
make test

