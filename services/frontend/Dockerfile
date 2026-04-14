FROM m.daocloud.io/docker.io/library/golang:1.26.1-alpine AS builder

WORKDIR /src
COPY shared/gen ./shared/gen
COPY services/frontend ./services/frontend

WORKDIR /src/services/frontend
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/frontend ./cmd/main.go

FROM m.daocloud.io/gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/frontend /app/frontend

EXPOSE 8080
ENTRYPOINT ["/app/frontend"]
