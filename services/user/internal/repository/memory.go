package repository

import "sync"

type User struct {
	ID       string
	Provider string
	OpenID   string
	Nickname string
	Avatar   string
	Email    string
}

type MemoryRepository struct {
	mu         sync.RWMutex
	byID       map[string]User
	byExternal map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:       make(map[string]User),
		byExternal: make(map[string]string),
	}
}

func makeExternalKey(provider, openID string) string {
	return provider + ":" + openID
}

func (r *MemoryRepository) FindByExternal(provider, openID string) (User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, ok := r.byExternal[makeExternalKey(provider, openID)]
	if !ok {
		return User{}, false
	}

	u, ok := r.byID[userID]
	if !ok {
		return User{}, false
	}

	return u, true
}

func (r *MemoryRepository) FindByID(userID string) (User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.byID[userID]
	return u, ok
}

func (r *MemoryRepository) Save(user User) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[user.ID] = user
	r.byExternal[makeExternalKey(user.Provider, user.OpenID)] = user.ID
}

func (r *MemoryRepository) UpdateProfile(userID, nickname, avatar, email string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.byID[userID]
	if !ok {
		return false
	}

	u.Nickname = nickname
	u.Avatar = avatar
	u.Email = email
	r.byID[userID] = u
	return true
}
