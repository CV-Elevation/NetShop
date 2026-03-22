package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"netshop/services/user/internal/handler"
	"netshop/services/user/internal/repository"
	"netshop/services/user/internal/service"
)

func main() {
	addr := os.Getenv("USER_GRPC_ADDR")
	if addr == "" {
		addr = ":50051"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	//存在内存里，开发阶段快速验证
	repo := repository.NewMemoryRepository()
	userSvc := service.NewUserService(repo)

	grpcServer := grpc.NewServer()
	handler.Register(grpcServer, userSvc)

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
