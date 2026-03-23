package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"netshop/services/ad/internal/handler"
	"netshop/services/ad/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	addr := os.Getenv("AD_GRPC_ADDR")
	if addr == "" {
		addr = ":50055"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	svc := service.NewAdService()
	grpcServer := grpc.NewServer()
	handler.Register(grpcServer, svc)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("ad grpc service listening at %s", addr)
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
	log.Println("ad grpc service stopped")
}
