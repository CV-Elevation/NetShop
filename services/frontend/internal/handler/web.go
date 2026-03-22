package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"html/template"
	"log"
	"net/http"
	"time"

	"netshop/services/frontend/internal/client"
	"netshop/services/frontend/internal/config"
	"netshop/services/frontend/internal/middleware"
	"netshop/services/frontend/internal/oauth"
	view "netshop/services/frontend/internal/template"
	"netshop/services/frontend/internal/token"
)

const oauthStateCookie = "oauth_state"

type WebHandler struct {
	cfg         config.Config
	tmpl        *template.Template
	oauthClient *oauth.GitHubClient
	tokens      *token.Manager
	userClient  *client.UserServiceClient
	emailClient *client.EmailServiceClient
}

type loginPageData struct {
	GitHubLoginURL string
	Message        string
}

type homePageData struct {
	Nickname string
	Email    string
	UserID   string
}

func NewWebHandler(
	cfg config.Config,
	oauthClient *oauth.GitHubClient,
	tokens *token.Manager,
	userClient *client.UserServiceClient,
	emailClient *client.EmailServiceClient,
) (*WebHandler, error) {
	tmpl, err := view.Parse()
	if err != nil {
		return nil, err
	}

	return &WebHandler{
		cfg:         cfg,
		tmpl:        tmpl,
		oauthClient: oauthClient,
		tokens:      tokens,
		userClient:  userClient,
		emailClient: emailClient,
	}, nil
}

func (h *WebHandler) Register(mux *http.ServeMux, authMiddleware *middleware.AuthMiddleware) {
	mux.HandleFunc("/healthz", h.healthz)
	mux.HandleFunc("/login", h.loginPage)
	mux.HandleFunc("/auth/github/login", h.githubLogin)
	mux.HandleFunc("/auth/github/callback", h.githubCallback)
	//这里是把内层的裸函数进行了包装，包装上了登录拦截的逻辑
	mux.Handle("/", authMiddleware.RequireAuth(http.HandlerFunc(h.homePage)))
	mux.Handle("/logout", authMiddleware.RequireAuth(http.HandlerFunc(h.logout)))
}

// 健康检测接口
func (h *WebHandler) healthz(w http.ResponseWriter, r *http.Request) {
	//请求方式为GET
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// 登录页面
func (h *WebHandler) loginPage(w http.ResponseWriter, r *http.Request) {
	//用户提示消息
	message := r.URL.Query().Get("msg")
	if err := h.tmpl.ExecuteTemplate(w, "login.html", loginPageData{
		GitHubLoginURL: "/auth/github/login",
		Message:        message,
	}); err != nil {
		http.Error(w, "render login failed", http.StatusInternalServerError)
	}
}

// github登录的逻辑
func (h *WebHandler) githubLogin(w http.ResponseWriter, r *http.Request) {
	//检查是否正确配置了OAuth
	if !h.oauthClient.IsConfigured() {
		http.Redirect(w, r, "/login?msg=github+oauth+is+not+configured", http.StatusFound)
		return
	}
	//生成状态码，防止CSRF攻击
	state, err := generateState()
	if err != nil {
		http.Error(w, "generate oauth state failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(10 * time.Minute),
	})
	http.Redirect(w, r, h.oauthClient.AuthURL(state), http.StatusFound)
}

func (h *WebHandler) githubCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(oauthStateCookie)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		http.Redirect(w, r, "/login?msg=invalid+oauth+state", http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/login?msg=missing+oauth+code", http.StatusFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	log.Printf("Received GitHub OAuth code: %s", code)
	accessToken, err := h.oauthClient.ExchangeCode(ctx, code)
	if err != nil {
		log.Printf("exchange github code failed: %v", err)
		http.Redirect(w, r, "/login?msg=github+authorization+failed", http.StatusFound)
		return
	}

	githubUser, err := h.oauthClient.FetchUser(ctx, accessToken)
	if err != nil {
		log.Printf("fetch github user failed: %v", err)
		http.Redirect(w, r, "/login?msg=fetch+github+profile+failed", http.StatusFound)
		return
	}
	log.Printf("github oauth user resolved: openid=%s nickname=%s", githubUser.ID, githubUser.Nickname)
	//grpc调用user服务
	userResp, err := h.userClient.LoginOrRegister(ctx, client.LoginOrRegisterRequest{
		Provider: "github",
		OpenID:   githubUser.ID,
		Nickname: githubUser.Nickname,
		Avatar:   githubUser.AvatarURL,
		Email:    githubUser.Email,
	})
	if err != nil {
		log.Printf("user service login/register failed: %v", err)
		http.Redirect(w, r, "/login?msg=user+service+failed", http.StatusFound)
		return
	}
	//发送欢迎邮件
	if userResp.IsNew && githubUser.Email != "" {
		if err := h.emailClient.SendWelcome(ctx, client.SendWelcomeRequest{
			UserID:   userResp.UserID,
			Email:    githubUser.Email,
			Nickname: githubUser.Nickname,
		}); err != nil {
			log.Printf("send welcome email failed: %v", err)
		}
	}

	signedAccess, signedRefresh, err := h.tokens.IssuePair(userResp.UserID, githubUser.Email, githubUser.Nickname)
	if err != nil {
		http.Error(w, "issue jwt failed", http.StatusInternalServerError)
		return
	}
	//签发两个token
	h.tokens.SetAuthCookies(w, signedAccess, signedRefresh)

	// Clear one-time state cookie after callback is completed.
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode, //可以防止CSRF攻击
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
	//重定向回根目录
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *WebHandler) homePage(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	if err := h.tmpl.ExecuteTemplate(w, "home.html", homePageData{
		Nickname: user.Nickname,
		Email:    user.Email,
		UserID:   user.UserID,
	}); err != nil {
		http.Error(w, "render home failed", http.StatusInternalServerError)
	}
}

func (h *WebHandler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.tokens.ClearAuthCookies(w)
	http.Redirect(w, r, "/login?msg=logged+out", http.StatusFound)
}

func generateState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(buf)
	if state == "" {
		return "", errors.New("state is empty")
	}
	return state, nil
}
