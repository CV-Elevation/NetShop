package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	adpb "kuoz/netshop/platform/shared/proto/ad"
	aiassistantpb "kuoz/netshop/platform/shared/proto/aiassistant"
	cartpb "kuoz/netshop/platform/shared/proto/cart"
	commonpb "kuoz/netshop/platform/shared/proto/common"
	recommendpb "kuoz/netshop/platform/shared/proto/recommend"
	"netshop/services/frontend/internal/client"
	"netshop/services/frontend/internal/config"
	"netshop/services/frontend/internal/middleware"
	"netshop/services/frontend/internal/oauth"
	view "netshop/services/frontend/internal/template"
	"netshop/services/frontend/internal/token"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const oauthStateCookie = "oauth_state"

type WebHandler struct {
	cfg               config.Config
	tmpl              *template.Template
	oauthClient       *oauth.GitHubClient
	tokens            *token.Manager
	userClient        *client.UserServiceClient
	emailClient       *client.EmailServiceClient
	productClient     *client.ProductServiceClient
	adClient          *client.AdServiceClient
	recommendClient   *client.RecommendServiceClient
	cartClient        *client.CartServiceClient
	aiAssistantClient *client.AIAssistantServiceClient
}

type loginPageData struct {
	GitHubLoginURL string
	Message        string
}

type homePageData struct {
	Nickname string
	Email    string
	UserID   string

	Products          []productCard
	Recommendations   []productCard
	Ads               []adCard
	RecommendStrategy string
	Warnings          []string
}

type productCard struct {
	ID          string
	Name        string
	Description string
	Category    string
	Stock       int32
	Price       string
	ImageURL    string
	Rating      float32
	DetailURL   string
}

type productDetailPageData struct {
	Nickname string
	UserID   string
	Product  productCard
	Message  string
}

type cartPageData struct {
	Nickname   string
	UserID     string
	Items      []cartItemCard
	TotalPrice string
	TotalCount int32
	HasInvalid bool
}

type cartItemCard struct {
	ProductID   string
	Name        string
	ImageURL    string
	Quantity    int32
	Price       string
	Subtotal    string
	StockStatus string
	StockCount  int32
	Checked     bool
	ProductURL  string
}

type adCard struct {
	ID        string
	Title     string
	ImageURL  string
	TargetURL string
}

func NewWebHandler(
	cfg config.Config,
	oauthClient *oauth.GitHubClient,
	tokens *token.Manager,
	userClient *client.UserServiceClient,
	emailClient *client.EmailServiceClient,
	productClient *client.ProductServiceClient,
	adClient *client.AdServiceClient,
	recommendClient *client.RecommendServiceClient,
	cartClient *client.CartServiceClient,
	aiAssistantClient *client.AIAssistantServiceClient,
) (*WebHandler, error) {
	tmpl, err := view.Parse()
	if err != nil {
		return nil, err
	}

	return &WebHandler{
		cfg:               cfg,
		tmpl:              tmpl,
		oauthClient:       oauthClient,
		tokens:            tokens,
		userClient:        userClient,
		emailClient:       emailClient,
		productClient:     productClient,
		adClient:          adClient,
		recommendClient:   recommendClient,
		cartClient:        cartClient,
		aiAssistantClient: aiAssistantClient,
	}, nil
}

func (h *WebHandler) Register(mux *http.ServeMux, authMiddleware *middleware.AuthMiddleware) {
	mux.HandleFunc("/healthz", h.healthz)
	mux.HandleFunc("/login", h.loginPage)
	mux.HandleFunc("/auth/github/login", h.githubLogin)
	mux.HandleFunc("/auth/github/callback", h.githubCallback)
	//这里是把内层的裸函数进行了包装，包装上了登录拦截的逻辑
	mux.Handle("/products/", authMiddleware.RequireAuth(http.HandlerFunc(h.productDetailPage)))
	mux.Handle("/cart", authMiddleware.RequireAuth(http.HandlerFunc(h.cartPage)))
	mux.Handle("/cart/add", authMiddleware.RequireAuth(http.HandlerFunc(h.addToCart)))
	mux.Handle("/assistant/chat", authMiddleware.RequireAuth(http.HandlerFunc(h.aiAssistantChat)))
	mux.Handle("/", authMiddleware.RequireAuth(http.HandlerFunc(h.homePage)))
	mux.Handle("/logout", authMiddleware.RequireAuth(http.HandlerFunc(h.logout)))
}

type aiAssistantStreamResponse struct {
	Type     string        `json:"type"`
	Delta    string        `json:"delta,omitempty"`
	ToolCall string        `json:"tool_call,omitempty"`
	Products []productCard `json:"products,omitempty"`
	Done     bool          `json:"done,omitempty"`
}

func (h *WebHandler) aiAssistantChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	message := strings.TrimSpace(r.FormValue("message"))
	if message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
	defer cancel()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	err := h.aiAssistantClient.StreamChat(ctx, client.ChatRequest{
		SessionID: user.UserID,
		UserID:    user.UserID,
		Message:   message,
	}, func(chunk *aiassistantpb.ChatChunk) error {
		if chunk == nil {
			return nil
		}

		event := aiAssistantStreamResponse{Type: chunk.GetChunkType()}
		switch chunk.GetChunkType() {
		case "text":
			event.Delta = chunk.GetDelta()
		case "tool_status":
			if toolCall := chunk.GetToolCall(); toolCall != nil {
				summary := strings.TrimSpace(toolCall.GetSummary())
				if summary == "" {
					summary = toolCall.GetToolName()
				}
				event.ToolCall = summary
			}
		case "products":
			event.Products = mapProducts(chunk.GetProducts())
		case "done":
			event.Done = chunk.GetDone()
		default:
			event.Delta = chunk.GetDelta()
		}

		payload, marshalErr := json.Marshal(event)
		if marshalErr != nil {
			return marshalErr
		}
		if _, writeErr := w.Write(append(payload, '\n')); writeErr != nil {
			return writeErr
		}
		flusher.Flush()
		return nil
	})
	if err != nil {
		log.Printf("aiassistant chat failed: %v", err)
		errorPayload, _ := json.Marshal(aiAssistantStreamResponse{
			Type:     "error",
			ToolCall: "assistant service unavailable",
			Done:     true,
		})
		_, _ = w.Write(append(errorPayload, '\n'))
		flusher.Flush()
		return
	}
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
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	data := homePageData{
		Nickname: user.Nickname,
		Email:    user.Email,
		UserID:   user.UserID,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		resp, err := h.recommendClient.GetRecommendations(ctx, client.GetRecommendationsRequest{
			UserID: user.UserID,
			Scene:  recommendpb.Scene_SCENE_HOMEPAGE,
			Limit:  6,
		})
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			log.Printf("recommend service failed: %v", err)
			data.Warnings = append(data.Warnings, "推荐服务暂时不可用，已降级展示")
			return
		}
		data.RecommendStrategy = resp.Strategy
		data.Recommendations = mapProducts(resp.Items)
	}()

	go func() {
		defer wg.Done()
		resp, err := h.productClient.ListProducts(ctx, client.ListProductsRequest{
			Category: "",
			Page:     1,
			PageSize: 8,
		})
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			log.Printf("product service failed: %v", err)
			data.Warnings = append(data.Warnings, "商品服务暂时不可用，商品列表为空")
			return
		}
		data.Products = mapProducts(resp.Items)
	}()

	go func() {
		defer wg.Done()
		items, err := h.adClient.GetAds(ctx, client.GetAdsRequest{
			UserID: user.UserID,
			Slot:   adpb.AdSlot_AD_SLOT_BANNER,
			Limit:  2,
		})
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			log.Printf("ad service failed: %v", err)
			data.Warnings = append(data.Warnings, "广告服务暂时不可用")
			return
		}
		data.Ads = mapAds(items)
	}()

	wg.Wait()

	if err := h.tmpl.ExecuteTemplate(w, "home.html", data); err != nil {
		http.Error(w, "render home failed", http.StatusInternalServerError)
	}
}

func (h *WebHandler) productDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	productID := strings.TrimPrefix(r.URL.Path, "/products/")
	productID, err := url.PathUnescape(productID)
	if err != nil || productID == "" {
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	product, err := h.productClient.GetProduct(ctx, client.GetProductRequest{ID: productID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			http.NotFound(w, r)
			return
		}
		log.Printf("product detail fetch failed: %v", err)
		http.Error(w, "load product failed", http.StatusBadGateway)
		return
	}

	data := productDetailPageData{
		Nickname: user.Nickname,
		UserID:   user.UserID,
		Product:  mapOneProduct(product),
		Message:  r.URL.Query().Get("msg"),
	}

	if err := h.tmpl.ExecuteTemplate(w, "product_detail.html", data); err != nil {
		http.Error(w, "render product detail failed", http.StatusInternalServerError)
	}
}

func (h *WebHandler) addToCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	productID := strings.TrimSpace(r.FormValue("product_id"))
	if productID == "" {
		http.Error(w, "product_id is required", http.StatusBadRequest)
		return
	}

	quantity, err := strconv.Atoi(strings.TrimSpace(r.FormValue("quantity")))
	if err != nil || quantity <= 0 {
		http.Redirect(w, r, "/products/"+url.PathEscape(productID)+"?msg=Invalid+quantity", http.StatusFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err = h.cartClient.AddItem(ctx, client.AddCartItemRequest{
		UserID:    user.UserID,
		ProductID: productID,
		Quantity:  int32(quantity),
	})
	if err != nil {
		log.Printf("add to cart failed: %v", err)
		http.Redirect(w, r, "/products/"+url.PathEscape(productID)+"?msg=Add+to+cart+failed", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/products/"+url.PathEscape(productID)+"?msg=Added+to+cart", http.StatusFound)
}

func (h *WebHandler) cartPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/cart" {
		http.NotFound(w, r)
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.cartClient.GetCart(ctx, client.GetCartRequest{UserID: user.UserID})
	if err != nil {
		log.Printf("load cart failed: %v", err)
		http.Error(w, "load cart failed", http.StatusBadGateway)
		return
	}

	data := cartPageData{
		Nickname:   user.Nickname,
		UserID:     user.UserID,
		Items:      mapCartItems(resp.Items),
		TotalPrice: formatPrice(resp.TotalPrice.GetAmount(), resp.TotalPrice.GetCurrency()),
		TotalCount: resp.TotalCount,
		HasInvalid: resp.HasInvalid,
	}

	if err := h.tmpl.ExecuteTemplate(w, "cart.html", data); err != nil {
		http.Error(w, "render cart failed", http.StatusInternalServerError)
	}
}

func mapProducts(items []*commonpb.Product) []productCard {
	result := make([]productCard, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, mapOneProduct(item))
	}
	return result
}

func mapOneProduct(item *commonpb.Product) productCard {
	if item == nil {
		return productCard{}
	}
	return productCard{
		ID:          item.GetId(),
		Name:        item.GetName(),
		Description: item.GetDescription(),
		Category:    item.GetCategory(),
		Stock:       item.GetStock(),
		Price:       formatPrice(item.GetPrice().GetAmount(), item.GetPrice().GetCurrency()),
		ImageURL:    item.GetImageUrl(),
		Rating:      item.GetRating(),
		DetailURL:   "/products/" + url.PathEscape(item.GetId()),
	}
}

func mapCartItems(items []*cartpb.CartItem) []cartItemCard {
	result := make([]cartItemCard, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}

		productID := item.GetProductId()
		result = append(result, cartItemCard{
			ProductID:   productID,
			Name:        item.GetName(),
			ImageURL:    item.GetImageUrl(),
			Quantity:    item.GetQuantity(),
			Price:       formatPrice(item.GetPrice().GetAmount(), item.GetPrice().GetCurrency()),
			Subtotal:    formatPrice(item.GetSubtotal().GetAmount(), item.GetSubtotal().GetCurrency()),
			StockStatus: stockStatusLabel(item.GetStockStatus()),
			StockCount:  item.GetStockCount(),
			Checked:     item.GetChecked(),
			ProductURL:  "/products/" + url.PathEscape(productID),
		})
	}
	return result
}

func stockStatusLabel(status cartpb.StockStatus) string {
	switch status {
	case cartpb.StockStatus_IN_STOCK:
		return "In stock"
	case cartpb.StockStatus_LOW_STOCK:
		return "Low stock"
	case cartpb.StockStatus_INSUFFICIENT:
		return "Insufficient stock"
	case cartpb.StockStatus_OUT_OF_STOCK:
		return "Out of stock"
	default:
		return "Unknown"
	}
}

func mapAds(items []*adpb.Ad) []adCard {
	result := make([]adCard, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, adCard{
			ID:        item.GetId(),
			Title:     item.GetTitle(),
			ImageURL:  item.GetImageUrl(),
			TargetURL: item.GetTargetUrl(),
		})
	}
	return result
}

func formatPrice(amount int64, currency string) string {
	if currency == "" {
		currency = "CNY"
	}
	return fmt.Sprintf("%s %.2f", currency, float64(amount)/100)
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
