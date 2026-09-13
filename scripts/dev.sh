#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

mkdir -p /tmp/sports-logs

cleanup() {
  echo
  echo "stopping services..."
  kill 0 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "starting scoring, commentary, gateway, ingestion"
go run ./cmd/scoring    > /tmp/sports-logs/scoring.log 2>&1 &
go run ./cmd/commentary > /tmp/sports-logs/commentary.log 2>&1 &
go run ./cmd/gateway    > /tmp/sports-logs/gateway.log 2>&1 &
sleep 1
go run ./cmd/ingestion  > /tmp/sports-logs/ingestion.log 2>&1 &
wait
