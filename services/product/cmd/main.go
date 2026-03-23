package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"netshop/services/product/internal/handler"
	"netshop/services/product/internal/repository"
	"netshop/services/product/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// ── 数据库连接 ────────────────────────────────────────────
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://netshop:secret@localhost:5432/netshop?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("connect db failed: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("ping db failed: %v", err)
	}
	log.Println("database connected")

	// ── 服务初始化 ────────────────────────────────────────────
	addr := os.Getenv("PRODUCT_GRPC_ADDR")
	if addr == "" {
		addr = ":50053"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	repo := repository.NewPostgresRepository(pool)
	productSvc := service.NewProductService(repo)
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	handler.Register(grpcServer, productSvc)

	// ── 启动 & 优雅退出 ───────────────────────────────────────
	go func() {
		log.Printf("product grpc service listening at %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		grpcServer.Stop()
	}
	log.Println("product grpc service stopped")
}
