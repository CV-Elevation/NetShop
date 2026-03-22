package repository

import (
	"sync"

	emailpb "kuoz/netshop/platform/shared/proto/email"
)

type Notification struct {
	ID           string
	UserID       string
	Email        string
	Type         emailpb.NotificationType
	Data         *emailpb.NotificationData
	Status       emailpb.DeliveryStatus
	SentAt       int64
	ErrorMessage string
}

type MemoryRepository struct {
	mu    sync.RWMutex
	store map[string]Notification
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{store: make(map[string]Notification)}
}

func (r *MemoryRepository) Save(notification Notification) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[notification.ID] = notification
}

func (r *MemoryRepository) FindByID(id string) (Notification, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	n, ok := r.store[id]
	return n, ok
}

func (r *MemoryRepository) UpdateStatus(id string, status emailpb.DeliveryStatus, sentAt int64, errMsg string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	n, ok := r.store[id]
	if !ok {
		return false
	}

	n.Status = status
	n.SentAt = sentAt
	n.ErrorMessage = errMsg
	r.store[id] = n
	return true
}
