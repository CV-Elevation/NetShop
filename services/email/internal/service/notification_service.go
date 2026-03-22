package service

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	emailpb "kuoz/netshop/platform/shared/proto/email"
	"netshop/services/email/internal/repository"
)

type NotificationService struct {
	repo      *repository.MemoryRepository
	idCounter uint64
}

func NewNotificationService(repo *repository.MemoryRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) SendNotification(_ context.Context, req *emailpb.SendNotificationRequest) (string, error) {
	if req.GetEmail() == "" {
		return "", status.Error(codes.InvalidArgument, "email is required")
	}
	if req.GetType() == emailpb.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED {
		return "", status.Error(codes.InvalidArgument, "notification type is required")
	}
	if err := validatePayload(req); err != nil {
		return "", err
	}

	notificationID := fmt.Sprintf("noti_%d", atomic.AddUint64(&s.idCounter, 1))
	// s.repo.Save(repository.Notification{
	// 	ID:     notificationID,
	// 	UserID: req.GetUserId(),
	// 	Email:  req.GetEmail(),
	// 	Type:   req.GetType(),
	// 	Data:   req.GetData(),
	// 	Status: emailpb.DeliveryStatus_DELIVERY_STATUS_PENDING,
	// })
	//异步发送
	go s.process(notificationID, req)
	return notificationID, nil
}

func (s *NotificationService) GetNotificationStatus(_ context.Context, notificationID string) (*emailpb.NotificationStatus, error) {
	if notificationID == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}

	n, ok := s.repo.FindByID(notificationID)
	if !ok {
		return nil, status.Error(codes.NotFound, "notification not found")
	}

	return &emailpb.NotificationStatus{
		NotificationId: n.ID,
		Status:         n.Status,
		SentAt:         n.SentAt,
		ErrorMessage:   n.ErrorMessage,
	}, nil
}

func (s *NotificationService) process(notificationID string, req *emailpb.SendNotificationRequest) {
	time.Sleep(120 * time.Millisecond)

	log.Printf("send notification id=%s type=%s to=%s user_id=%s", notificationID, req.GetType().String(), req.GetEmail(), req.GetUserId())
	// s.repo.UpdateStatus(notificationID, emailpb.DeliveryStatus_DELIVERY_STATUS_SENT, time.Now().Unix(), "")
}

func validatePayload(req *emailpb.SendNotificationRequest) error {
	data := req.GetData()
	if data == nil {
		return status.Error(codes.InvalidArgument, "data is required")
	}

	switch req.GetType() {
	case emailpb.NotificationType_NOTIFICATION_TYPE_ORDER_PLACED:
		if data.GetOrder() == nil {
			return status.Error(codes.InvalidArgument, "order payload is required for ORDER_PLACED")
		}
	case emailpb.NotificationType_NOTIFICATION_TYPE_WELCOME:
		if data.GetWelcome() == nil {
			return status.Error(codes.InvalidArgument, "welcome payload is required for WELCOME")
		}
	case emailpb.NotificationType_NOTIFICATION_TYPE_REFUND_PROCESSED:
		if data.GetRefund() == nil {
			return status.Error(codes.InvalidArgument, "refund payload is required for REFUND_PROCESSED")
		}
	default:
		return status.Error(codes.InvalidArgument, "unknown notification type")
	}

	return nil
}
