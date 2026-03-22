package client

import (
	"context"

	emailpb "kuoz/netshop/platform/shared/proto/email"
)

type EmailServiceClient struct {
	grpcClient emailpb.EmailServiceClient
}

type SendWelcomeRequest struct {
	UserID   string
	Email    string
	Nickname string
}

func NewEmailServiceClient(grpcClient emailpb.EmailServiceClient) *EmailServiceClient {
	return &EmailServiceClient{grpcClient: grpcClient}
}

func (c *EmailServiceClient) SendWelcome(ctx context.Context, req SendWelcomeRequest) error {
	_, err := c.grpcClient.SendNotification(ctx, &emailpb.SendNotificationRequest{
		UserId: req.UserID,
		Email:  req.Email,
		Type:   emailpb.NotificationType_NOTIFICATION_TYPE_WELCOME,
		Data: &emailpb.NotificationData{
			Payload: &emailpb.NotificationData_Welcome{
				Welcome: &emailpb.WelcomeNotification{Username: req.Nickname},
			},
		},
	})
	return err
}
