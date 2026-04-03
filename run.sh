#!/usr/bin/env bash
set -e

(cd services/user && go run ./cmd/main.go) &
(cd services/email && go run ./cmd/main.go) &
(cd services/product && go run ./cmd/main.go) &
(cd services/recommend && go run ./cmd/main.go) &
(cd services/ad && go run ./cmd/main.go) &
(cd services/cart && go run ./cmd/main.go) &
(cd services/aiassistant && go run ./cmd/main.go) &
sleep 2
(cd services/frontend && go run ./cmd/main.go) &

trap 'pids=$(jobs -pr); [ -n "$pids" ] && kill -TERM $pids; wait; exit 0' INT TERM
wait