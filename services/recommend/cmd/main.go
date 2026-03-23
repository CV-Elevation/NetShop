package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"netshop/services/recommend/internal/handler"
	"netshop/services/recommend/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	addr := os.Getenv("RECOMMEND_GRPC_ADDR")
	if addr == "" {
		addr = ":50054"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	svc := service.NewRecommendService()
	grpcServer := grpc.NewServer()
	handler.Register(grpcServer, svc)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("recommend grpc service listening at %s", addr)
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
	log.Println("recommend grpc service stopped")
}
