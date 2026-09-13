#  Sports Live — Real-Time Cricket Scoring System

A **multi-service, event-driven real-time cricket platform** built with Go.

The system processes ball-by-ball events, maintains live match state, generates automated commentary, persists historical data, and broadcasts real-time updates to connected web clients.

---

##  Overview

**Sports Live** follows a microservice architecture where independent Go services communicate asynchronously through **NATS JetStream and Core NATS**.

The platform separates:

* **Real-time state** — Redis
* **Persistent historical data** — PostgreSQL
*  **Event streaming** — NATS JetStream
*  **Commentary generation** — Dedicated commentary service
*  **Client delivery** — REST API + WebSockets

### High-Level Architecture

<img width="1223" height="1286" alt="Sports Live Architecture" src="https://github.com/user-attachments/assets/9107fe03-c27b-4ace-b48e-515089cc60e9" />

---

#  End-to-End Data Flow

```text
                    ┌─────────────────────────────┐
                    │     Ingestion / Simulator   │
                    │  Mock Feed / REST API       │
                    └──────────────┬──────────────┘
                                   │
                                   │ sports.events.normalized
                                   ▼
                    ┌─────────────────────────────┐
                    │       Scoring Engine        │
                    │                             │
                    │  • Cricket state machine    │
                    │  • Score calculation        │
                    │  • Player statistics        │
                    │  • Strike rotation          │
                    └───────┬───────────┬─────────┘
                            │           │
                 match.ball │           │ match.state
                            ▼           ▼
                 ┌──────────────┐   ┌──────────────┐
                 │  Commentary  │   │    Redis     │
                 │   Generator  │   │  Match State │
                 └──────┬───────┘   └──────────────┘
                        │
                        │ sports.commentary.<id>
                        ▼
                 ┌─────────────────────────┐
                 │ API Gateway + WebSocket │
                 │          Hub            │
                 └────────────┬────────────┘
                              │
                       WebSocket Fanout
                              │
                              ▼
                 ┌─────────────────────────┐
                 │   Frontend Dashboards   │
                 │                         │
                 │ index.html / admin.html │
                 │       / view.html       │
                 └─────────────────────────┘

                     ┌────────────────┐
                     │   PostgreSQL   │
                     │                │
                     │ Historical     │
                     │ ball-by-ball   │
                     │ persistence    │
                     └────────────────┘
```

---

#  Core Services

| Service              | Responsibility                                      |
| -------------------- | --------------------------------------------------- |
| **Ingestion**        | Receives or generates ball delivery events          |
| **Scoring**          | Calculates and maintains the live cricket state     |
| **Commentary**       | Converts ball events into human-readable commentary |
| **Gateway**          | Provides REST APIs and WebSocket fanout             |
| **Redis**            | Stores the latest match state for low-latency reads |
| **PostgreSQL**       | Persists historical ball-by-ball events             |
| **NATS / JetStream** | Provides asynchronous event streaming               |

---

# Event Flow

### 1. Ingestion

Ball delivery events enter the system through:

* Mock simulator
* REST API
* Future external feed adapters

The ingestion service publishes normalized events to:

```text
sports.events.normalized
```

inside the `EVENTS` JetStream stream.

---

### 2. Scoring Engine

The scoring service consumes:

```text
sports.events.normalized
```

using a durable JetStream consumer named:

```text
scoring
```

Each `BallDelivered` event is passed to the in-memory scoring engine.

The engine calculates:

* Total runs
* Wickets
* Overs
* Ball number
* Extra runs
* Batter statistics
* Bowler statistics
* Strike rotation
* Partnership statistics

The resulting events are published to:

```text
sports.match.<ID>.state
sports.match.<ID>.ball
```

The scoring service also:

1. Updates the latest match state in Redis
2. Persists the raw ball event in PostgreSQL

---

### 3. Commentary Generator

The commentary service consumes:

```text
sports.match.<ID>.ball
```

using a durable consumer named:

```text
commentary
```

It converts ball events into natural-language commentary.

For example:

```text
0.1 Bumrah to Kohli, FOUR
```

or:

```text
0.4 Bumrah to Kohli, OUT caught
```

The generated commentary is published to:

```text
sports.commentary.<ID>
```

through Core NATS.

---

### 4. API Gateway & WebSocket Hub

The gateway subscribes to:

```text
sports.match.*.>
sports.commentary.*
```

Incoming NATS messages are wrapped in a standard WebSocket envelope:

```json
{
  "type": "ball",
  "match_id": "match-001",
  "payload": {}
}
```

The WebSocket Hub maintains rooms for individual matches.

Connected clients receive real-time updates through WebSocket fanout.

---

# Data Storage

## Redis

Redis stores the latest match state for fast access.

Example key:

```text
match:<id>:state
```

Match indexes are maintained using:

```text
matches:index
```

The match state has a **24-hour TTL**.

Redis is primarily used for:

* ⚡ Fast match-state reads
* Current scoreboard data
* Match indexing

---

## PostgreSQL

PostgreSQL provides persistent historical storage.

The main tables are:

```text
matches
balls
```

The `balls` table stores:

* Match ID
* Innings ID
* Over number
* Ball number
* Batsman
* Bowler
* Runs
* Extras
* Wicket information
* Delivery type
* Delivery timestamp

Ball events are persisted using an idempotent insert:

```sql
ON CONFLICT (event_id) DO NOTHING
```

This prevents duplicate event persistence.

---

# NATS Architecture

NATS is used as the communication backbone between services.

### Important subjects

```text
sports.events.normalized
sports.match.<ID>.state
sports.match.<ID>.ball
sports.commentary.<ID>
```

### JetStream Streams

```text
EVENTS
MATCH_STATE
MATCH_BALLS
```

JetStream provides durable event consumption and persistence for the important event-processing paths.

Core NATS is used for lightweight real-time fanout, particularly for commentary and gateway subscriptions.

---

# 🏗️ Architecture Patterns

## Event-Driven Architecture

Services communicate asynchronously through NATS subjects rather than directly depending on each other.

```text
Producer → NATS → Consumer
```

This keeps the services loosely coupled.

---

## Domain-Driven State Machine

The scoring engine is implemented as a pure Go state machine:

```text
internal/scoring/engine.go
```

The engine performs cricket-rule calculations without direct I/O side effects.

This makes the core scoring logic easier to test independently.

---

## CQRS / Event-Sourcing Hybrid

State transitions are driven by `BallDelivered` events.

```text
BallDelivered Event
       │
       ▼
Scoring Engine
       │
       ├──► Redis → Current State
       │
       ├──► PostgreSQL → Historical Events
       │
       └──► NATS → Real-Time Consumers
```

Read models are served from Redis snapshots and real-time WebSocket feeds, while historical writes are persisted to PostgreSQL.

---

## WebSocket Room Hub

The gateway uses a concurrent room-based WebSocket Hub.

```text
Hub
 ├── match-001
 │    ├── Client A
 │    └── Client B
 │
 ├── match-002
 │    └── Client C
 │
 └── all
      └── Global Clients
```

Each connection uses separate `readPump` and `writePump` goroutines.

---

## Adapter Pattern

Ingestion is abstracted behind:

```go
type FeedAdapter interface {
    Connect()
    ReadEvents()
    Close()
}
```

This allows the mock feed to be replaced by a real external provider later.

---

# 📁 Project Structure

```text
sports/
│
├── cmd/                              # Service entrypoints
│   ├── ingestion/
│   │   └── main.go                  # Data feed simulator / ingest runner
│   ├── scoring/
│   │   └── main.go                  # State machine processing engine
│   ├── commentary/
│   │   └── main.go                  # Text commentary engine
│   └── gateway/
│       └── main.go                  # HTTP API + WebSocket server
│
├── internal/                         # Domain logic & services
│   ├── commentary/                   # Commentary generation
│   ├── events/                       # NATS event contracts
│   ├── gateway/                      # WebSocket Hub + NATS router
│   ├── ingestion/                    # Feed abstraction + mock feed
│   ├── models/                       # Match / innings / statistics
│   └── scoring/                      # Scoring state engine
│
├── pkg/                              # Reusable infrastructure
│   ├── db/                           # PostgreSQL client + schema
│   ├── nats/                         # NATS / JetStream wrapper
│   └── redis/                        # Redis client wrapper
│
├── web/                              # Static frontend
│   ├── admin.html                    # Admin & ball simulator
│   ├── index.html                    # Main dashboard
│   └── view.html                     # Dedicated live score view
│
├── scripts/
│   └── dev.sh                        # Local multi-process development
│
├── docker-compose.yml                # Container topology
├── Dockerfile                        # Multi-stage Go build
└── Makefile                          # Development commands
```

---

# 🔍 Important Files

### `cmd/ingestion/main.go`

Entry point for the ingestion service.

It checks:

```text
ENABLE_MOCK_FEED
```

When enabled, the service starts the mock feed simulator.

---

### `cmd/scoring/main.go`

Starts the scoring engine and connects it to:

* NATS
* Redis
* PostgreSQL

It creates the scoring service and starts event processing.

---

### `internal/scoring/engine.go`

The core cricket state machine.

Responsible for:

```text
Runs
Wickets
Overs
Balls
Extras
Batting statistics
Bowling statistics
Strike rotation
Partnerships
```

---

### `internal/scoring/service.go`

Connects the scoring engine with infrastructure.

For each event it:

```text
NATS Event
    ↓
Engine.Apply()
    ↓
Updated MatchState
    ├──► NATS
    ├──► Redis
    └──► PostgreSQL
```

---

### `internal/commentary/templates.go`

Contains the logic that converts ball data into human-readable commentary.

Examples include:

```text
FOUR
SIX
WICKET
WIDE
NO-BALL
NORMAL DELIVERY
```

---

### `internal/gateway/hub.go`

Maintains WebSocket clients and match rooms.

It provides:

```go
Register()
Unregister()
Broadcast()
Run()
```

The room structure is conceptually:

```go
map[matchID]map[*Client]struct{}
```

---

### `internal/gateway/subscriber.go`

Bridges NATS messages into the WebSocket Hub.

It:

1. Subscribes to NATS
2. Parses the subject
3. Extracts the match ID
4. Wraps the message in an envelope
5. Broadcasts it to the appropriate room

---

# 🖥️ Frontend

The project contains three static web interfaces.

### `index.html`

Main match dashboard featuring:

* Live score
* Overs progress
* Commentary timeline
* Scorecard
* Wagon wheel
* WebSocket connection status

### `admin.html`

Administrative control panel for:

* Creating matches
* Submitting ball deliveries
* Selecting runs
* Adding extras
* Adding wickets
* Starting/stopping simulation

### `view.html`

Minimal live scoreboard designed for standalone or embedded viewing.

---

#  Tech Stack

| Category         | Technology             |
| ---------------- | ---------------------- |
| Language         | Go 1.26.5              |
| Message Broker   | NATS 2.11              |
| Streaming        | NATS JetStream         |
| Cache            | Redis 7                |
| Database         | PostgreSQL 16          |
| WebSockets       | Gorilla WebSocket      |
| Database Driver  | pgx / pgxpool          |
| Redis Client     | go-redis/v9            |
| Containerization | Docker                 |
| Orchestration    | Docker Compose         |
| Automation       | GNU Make + POSIX Shell |
| Frontend         | HTML / JavaScript      |

---

# Getting Started

## Prerequisites

Install:

* Docker
* Go 1.26+

---

## Run the Full Stack

Build and start all services:

```bash
make up
```

View logs:

```bash
make logs
```

Stop the stack:

```bash
make down
```

---

## Local Development

Run NATS, Redis, and PostgreSQL in Docker while running the Go services locally:

```bash
make dev
```

---

# Access the Application

Main dashboard:

```text
http://localhost:8080/
```

Admin / simulator:

```text
http://localhost:8080/admin
```

Dedicated live view:

```text
http://localhost:8080/view
```

---

# Docker Services

Docker Compose runs:

```text
NATS
Redis
PostgreSQL
Scoring
Commentary
Gateway
Ingestion
```

Default infrastructure ports:

| Service         |   Port |
| --------------- | -----: |
| Gateway         | `8080` |
| NATS            | `4222` |
| NATS Monitoring | `8222` |
| Redis           | `6379` |
| PostgreSQL      | `5432` |

---

#  Testing

The scoring engine contains unit tests covering:

* Cumulative score calculation
* Ball progression
* Boundary tracking
* Consecutive deliveries

Run tests with:

```bash
make test
```

---

# Development Commands

The Makefile provides shortcuts for:

```text
make nats
make ingestion
make scoring
make commentary
make gateway
make dev
make up
make down
make logs
make test
make build
```

---

# 📌 Design Goals

The system is designed around:

* ⚡ Low-latency live match updates
* 📨 Asynchronous event processing
* 🔌 Loosely coupled services
* 💾 Persistent ball-by-ball history
* 📊 Fast access to current match state
* 🌐 Real-time WebSocket fanout
* 🧪 Testable domain logic
* 🐳 Reproducible containerized deployment
* 🔄 Extensible ingestion adapters

---

# 🔮 Future Extensions

The ingestion adapter architecture makes it possible to replace the mock feed with:

```text
External REST Feed
       │
       ▼
FeedAdapter
       │
       ▼
NATS
       │
       ▼
Scoring Engine
```

This allows the same scoring pipeline to process real external cricket data without changing the core scoring engine.

---

---

<div align="center">

### Sports Live

**Real-time scoring • Event streaming • Live commentary • WebSocket fanout**

</div>
