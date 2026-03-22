package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"netshop/services/user/internal/handler"
	"netshop/services/user/internal/repository"
	"netshop/services/user/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
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

	// 验证连接是否正常
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("ping db failed: %v", err)
	}
	log.Println("database connected")

	// ── 服务初始化 ────────────────────────────────────────────
	addr := os.Getenv("USER_GRPC_ADDR")
	if addr == "" {
		addr = ":50051"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	repo := repository.NewPostgresRepository(pool) // 传入连接池
	userSvc := service.NewUserService(repo)
	grpcServer := grpc.NewServer()
	handler.Register(grpcServer, userSvc)

	// ── 启动 & 优雅退出 ───────────────────────────────────────
	go func() {
		log.Printf("user grpc service listening at %s", addr)
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
	log.Println("user grpc service stopped")
}
