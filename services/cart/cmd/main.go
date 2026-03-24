package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"netshop/services/cart/internal/handler"
	"netshop/services/cart/internal/repository"
	"netshop/services/cart/internal/service"

	productpb "kuoz/netshop/platform/shared/proto/product"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	// ── Redis 连接 ────────────────────────────────────────────
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	// ── Product 服务连接 ──────────────────────────────────────
	productAddr := os.Getenv("PRODUCT_GRPC_ADDR")
	if productAddr == "" {
		productAddr = "localhost:50053"
	}
	productConn, err := grpc.NewClient(productAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect product service failed: %v", err)
	}
	defer productConn.Close()
	productClient := productpb.NewProductServiceClient(productConn)

	// ── 服务初始化 ────────────────────────────────────────────
	addr := os.Getenv("CART_GRPC_ADDR")
	if addr == "" {
		addr = ":50056"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	repo := repository.NewRedisRepository(rdb)
	cartSvc := service.NewCartService(repo, productClient)
	grpcServer := grpc.NewServer()
	handler.Register(grpcServer, cartSvc)
	reflection.Register(grpcServer)

	// ── 启动 & 优雅退出 ───────────────────────────────────────
	go func() {
		log.Printf("cart grpc service listening at %s", addr)
		if grpcServer.Serve(lis); err != nil {
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
	log.Println("cart grpc service stopped")
}
