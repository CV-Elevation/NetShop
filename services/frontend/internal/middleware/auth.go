package middleware

import (
	"context"
	"net/http"

	"netshop/services/frontend/internal/token"
)

type ctxKey string

const userKey ctxKey = "authenticated_user"

type AuthenticatedUser struct {
	UserID   string
	Email    string
	Nickname string
}

type AuthMiddleware struct {
	tokens *token.Manager
}

func NewAuthMiddleware(tokens *token.Manager) *AuthMiddleware {
	return &AuthMiddleware{tokens: tokens}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := m.authenticate(w, r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		ctx := context.WithValue(r.Context(), userKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) authenticate(w http.ResponseWriter, r *http.Request) (AuthenticatedUser, bool) {
	accessCookie, err := r.Cookie(token.AccessCookieName)
	if err == nil && accessCookie.Value != "" {
		claims, accessErr := m.tokens.ParseAccess(accessCookie.Value)
		if accessErr == nil {
			return AuthenticatedUser{
				UserID:   claims.UserID,
				Email:    claims.Email,
				Nickname: claims.Nickname,
			}, true
		}
		if !token.IsExpired(accessErr) {
			return AuthenticatedUser{}, false
		}
	}
	//这里加入refresh-token本意是想存到redis，来管控token黑名单机制的，但是现阶段引入redis有点复杂，没什么意义
	refreshCookie, err := r.Cookie(token.RefreshCookieName)
	if err != nil || refreshCookie.Value == "" {
		return AuthenticatedUser{}, false
	}
	refreshClaims, refreshErr := m.tokens.ParseRefresh(refreshCookie.Value)
	if refreshErr != nil {
		return AuthenticatedUser{}, false
	}

	newAccess, newRefresh, issueErr := m.tokens.IssuePair(refreshClaims.UserID, refreshClaims.Email, refreshClaims.Nickname)
	if issueErr != nil {
		return AuthenticatedUser{}, false
	}
	m.tokens.SetAuthCookies(w, newAccess, newRefresh)

	return AuthenticatedUser{
		UserID:   refreshClaims.UserID,
		Email:    refreshClaims.Email,
		Nickname: refreshClaims.Nickname,
	}, true
}

func UserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(userKey).(AuthenticatedUser)
	return user, ok
}
