package handler

import (
	"context"

	"google.golang.org/grpc"

	emailpb "kuoz/netshop/platform/shared/proto/email"
	"netshop/services/email/internal/service"
)

type grpcServer struct {
	emailpb.UnimplementedEmailServiceServer
	svc *service.NotificationService
}

func Register(server *grpc.Server, svc *service.NotificationService) {
	emailpb.RegisterEmailServiceServer(server, &grpcServer{svc: svc})
}

func (s *grpcServer) SendNotification(ctx context.Context, req *emailpb.SendNotificationRequest) (*emailpb.SendNotificationResponse, error) {
	notificationID, err := s.svc.SendNotification(ctx, req)
	if err != nil {
		return nil, err
	}

	return &emailpb.SendNotificationResponse{NotificationId: notificationID}, nil
}

func (s *grpcServer) GetNotificationStatus(ctx context.Context, req *emailpb.GetNotificationStatusRequest) (*emailpb.NotificationStatus, error) {
	return s.svc.GetNotificationStatus(ctx, req.GetNotificationId())
}
