FROM m.daocloud.io/docker.io/library/golang:1.26.1-alpine AS builder

WORKDIR /src
COPY shared/gen ./shared/gen
COPY services/product ./services/product

WORKDIR /src/services/product
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/product ./cmd/main.go

FROM m.daocloud.io/gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/product /app/product

EXPOSE 50053
ENTRYPOINT ["/app/product"]
