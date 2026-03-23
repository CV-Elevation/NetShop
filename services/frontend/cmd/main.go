package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	adpb "kuoz/netshop/platform/shared/proto/ad"
	emailpb "kuoz/netshop/platform/shared/proto/email"
	productpb "kuoz/netshop/platform/shared/proto/product"
	recommendpb "kuoz/netshop/platform/shared/proto/recommend"
	userpb "kuoz/netshop/platform/shared/proto/user"
	"netshop/services/frontend/internal/client"
	"netshop/services/frontend/internal/config"
	"netshop/services/frontend/internal/handler"
	"netshop/services/frontend/internal/middleware"
	"netshop/services/frontend/internal/oauth"
	"netshop/services/frontend/internal/token"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()

	oauthClient := oauth.NewGitHubClient(cfg)
	tokenManager := token.NewManager(cfg)

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dialCancel()
	//连接user服务
	connUser, err := grpc.DialContext(
		dialCtx,
		cfg.UserServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("dial user service grpc failed: %v", err)
	}
	//连接email服务
	connEmail, err := grpc.DialContext(
		dialCtx,
		cfg.EmailServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		_ = connUser.Close()
		log.Fatalf("dial email service grpc failed: %v", err)
	}
	//连接product服务
	connProduct, err := grpc.DialContext(
		dialCtx,
		cfg.ProductServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		_ = connUser.Close()
		_ = connEmail.Close()
		log.Fatalf("dial product service grpc failed: %v", err)
	}
	//连接ad服务
	connAd, err := grpc.DialContext(
		dialCtx,
		cfg.AdServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		_ = connUser.Close()
		_ = connEmail.Close()
		_ = connProduct.Close()
		log.Fatalf("dial ad service grpc failed: %v", err)
	}
	//连接recommend服务
	connRecommend, err := grpc.DialContext(
		dialCtx,
		cfg.RecommendServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		_ = connUser.Close()
		_ = connEmail.Close()
		_ = connProduct.Close()
		_ = connAd.Close()
		log.Fatalf("dial recommend service grpc failed: %v", err)
	}

	userClient := client.NewUserServiceClient(userpb.NewUserServiceClient(connUser))
	emailClient := client.NewEmailServiceClient(emailpb.NewEmailServiceClient(connEmail))
	productClient := client.NewProductServiceClient(productpb.NewProductServiceClient(connProduct))
	adClient := client.NewAdServiceClient(adpb.NewAdServiceClient(connAd))
	recommendClient := client.NewRecommendServiceClient(recommendpb.NewRecommendServiceClient(connRecommend))
	//设置拦截器
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)

	//配置handler
	h, err := handler.NewWebHandler(cfg, oauthClient, tokenManager, userClient, emailClient, productClient, adClient, recommendClient)
	if err != nil {
		_ = connUser.Close()
		_ = connEmail.Close()
		_ = connProduct.Close()
		_ = connAd.Close()
		_ = connRecommend.Close()
		log.Fatalf("init handler failed: %v", err)
	}
	mux := http.NewServeMux()
	h.Register(mux, authMiddleware)

	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("frontend gateway listening at %s", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()
	//如果收到系统的关闭信号
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	//关闭 gRPC 连接
	if err := connUser.Close(); err != nil {
		log.Printf("close user grpc connection failed: %v", err)
	}
	if err := connEmail.Close(); err != nil {
		log.Printf("close email grpc connection failed: %v", err)
	}
	if err := connProduct.Close(); err != nil {
		log.Printf("close product grpc connection failed: %v", err)
	}
	if err := connAd.Close(); err != nil {
		log.Printf("close ad grpc connection failed: %v", err)
	}
	if err := connRecommend.Close(); err != nil {
		log.Printf("close recommend grpc connection failed: %v", err)
	}
}
