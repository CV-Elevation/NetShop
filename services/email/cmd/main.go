package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"netshop/services/email/internal/handler"
	"netshop/services/email/internal/repository"
	"netshop/services/email/internal/service"
)

func main() {
	addr := os.Getenv("EMAIL_GRPC_ADDR")
	if addr == "" {
		addr = ":50052"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	repo := repository.NewMemoryRepository()
	notificationSvc := service.NewNotificationService(repo)

	grpcServer := grpc.NewServer()
	handler.Register(grpcServer, notificationSvc)

	go func() {
		log.Printf("email grpc service listening at %s", addr)
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

	log.Println("email grpc service stopped")
}
