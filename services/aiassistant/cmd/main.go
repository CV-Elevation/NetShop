package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"netshop/services/aiassistant/internal/handler"
	"netshop/services/aiassistant/internal/repository"
	"netshop/services/aiassistant/internal/service"
	"netshop/services/aiassistant/internal/service/llm"

	productpb "kuoz/netshop/platform/shared/proto/product"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
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
	addr := os.Getenv("AIASSISTANT_GRPC_ADDR")
	if addr == "" {
		addr = ":50057"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	doubaoClient := llm.NewDoubaoClientFromEnv()
	intentClassifier := llm.NewLocalIntentClassifierFromEnv()

	kbDSN := os.Getenv("KNOWLEDGE_DB_DSN")
	if kbDSN == "" {
		kbDSN = "postgres://netshop:secret@localhost:5432/netshop?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	kbRepo, err := repository.NewKnowledgeBaseRepository(ctx, kbDSN, doubaoClient)
	if err != nil {
		log.Printf("knowledge base disabled: %v", err)
	}

	repo := repository.NewProductRepository(productClient, kbRepo)
	defer repo.Close()

	if err := repo.BuildKnowledgeBase(ctx); err != nil {
		log.Printf("knowledge base bootstrap skipped: %v", err)
	}

	aiAssistantSvc := service.NewAIAssistantService(repo, intentClassifier, doubaoClient)
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	handler.Register(grpcServer, aiAssistantSvc)

	// ── 启动 & 优雅退出 ───────────────────────────────────────
	go func() {
		log.Printf("aiassistant grpc service listening at %s", addr)
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
	log.Println("aiassistant grpc service stopped")
}
