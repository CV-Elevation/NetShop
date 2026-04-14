FROM m.daocloud.io/docker.io/library/golang:1.26.1-alpine AS builder

WORKDIR /src
COPY shared/gen ./shared/gen
COPY services/cart ./services/cart

WORKDIR /src/services/cart
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/cart ./cmd/main.go

FROM m.daocloud.io/gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/cart /app/cart

EXPOSE 50056
ENTRYPOINT ["/app/cart"]
