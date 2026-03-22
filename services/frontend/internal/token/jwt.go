package token

import (
	"errors"
	"net/http"
	"time"

	"netshop/services/frontend/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessCookieName  = "access_token"
	RefreshCookieName = "refresh_token"
)

type Claims struct {
	TokenType string `json:"token_type"`
	UserID    string `json:"uid"`
	Email     string `json:"email,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret       []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
	cookieSecure bool
}

func NewManager(cfg config.Config) *Manager {
	return &Manager{
		secret:       []byte(cfg.JWTSecret),
		accessTTL:    cfg.AccessTTL,
		refreshTTL:   cfg.RefreshTTL,
		cookieSecure: cfg.CookieSecure,
	}
}

func (m *Manager) IssuePair(userID, email, nickname string) (string, string, error) {
	now := time.Now()
	//生成access-token
	accessClaims := Claims{
		TokenType: "access",
		UserID:    userID,
		Email:     email,
		Nickname:  nickname,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}
	//生成refresh-token
	refreshClaims := Claims{
		TokenType: "refresh",
		UserID:    userID,
		Email:     email,
		Nickname:  nickname,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccess, err := accessToken.SignedString(m.secret)
	if err != nil {
		return "", "", err
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefresh, err := refreshToken.SignedString(m.secret)
	if err != nil {
		return "", "", err
	}
	return signedAccess, signedRefresh, nil
}

func (m *Manager) ParseAccess(token string) (*Claims, error) {
	claims, err := m.parse(token)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "access" {
		return nil, errors.New("invalid token type")
	}
	return claims, nil
}

func (m *Manager) ParseRefresh(token string) (*Claims, error) {
	claims, err := m.parse(token)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "refresh" {
		return nil, errors.New("invalid token type")
	}
	return claims, nil
}

func (m *Manager) parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func IsExpired(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired)
}

func (m *Manager) SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	accessExpiry := time.Now().Add(m.accessTTL)
	refreshExpiry := time.Now().Add(m.refreshTTL)

	http.SetCookie(w, &http.Cookie{
		Name:     AccessCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  accessExpiry,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  refreshExpiry,
	})
}

func (m *Manager) ClearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{AccessCookieName, RefreshCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   m.cookieSecure,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
		})
	}
}
