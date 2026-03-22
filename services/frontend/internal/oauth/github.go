package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"netshop/services/frontend/internal/config"
)

const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token"
	githubUserURL      = "https://api.github.com/user"
	githubEmailsURL    = "https://api.github.com/user/emails"
)

type GitHubUser struct {
	ID        string
	Nickname  string
	AvatarURL string
	Email     string
}

type GitHubClient struct {
	clientID     string
	clientSecret string
	callbackURL  string
	httpClient   *http.Client
}

func NewGitHubClient(cfg config.Config) *GitHubClient {
	return &GitHubClient{
		clientID:     cfg.GitHubClientID,
		clientSecret: cfg.GitHubClientSecret,
		callbackURL:  cfg.GitHubCallbackURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *GitHubClient) IsConfigured() bool {
	return c.clientID != "" && c.clientSecret != ""
}

func (c *GitHubClient) AuthURL(state string) string {
	query := url.Values{}
	query.Set("client_id", c.clientID)
	query.Set("redirect_uri", c.callbackURL)
	query.Set("state", state)
	query.Set("scope", "read:user user:email")
	return githubAuthorizeURL + "?" + query.Encode()
}

func (c *GitHubClient) ExchangeCode(ctx context.Context, code string) (string, error) {
	values := url.Values{}
	values.Set("client_id", c.clientID)
	values.Set("client_secret", c.clientSecret)
	values.Set("code", code)
	values.Set("redirect_uri", c.callbackURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("exchange github code failed: status=%d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Error != "" {
		return "", errors.New(tokenResp.Error)
	}
	if tokenResp.AccessToken == "" {
		return "", errors.New("github access token is empty")
	}
	return tokenResp.AccessToken, nil
}

func (c *GitHubClient) FetchUser(ctx context.Context, accessToken string) (GitHubUser, error) {
	user, err := c.fetchPrimaryUser(ctx, accessToken)
	if err != nil {
		return GitHubUser{}, err
	}
	if user.Email == "" {
		email, err := c.fetchPrimaryEmail(ctx, accessToken)
		if err == nil {
			user.Email = email
		}
	}
	return user, nil
}

func (c *GitHubClient) fetchPrimaryUser(ctx context.Context, accessToken string) (GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserURL, nil)
	if err != nil {
		return GitHubUser{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "netshop-frontend-gateway")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GitHubUser{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GitHubUser{}, fmt.Errorf("fetch github user failed: status=%d", resp.StatusCode)
	}

	var payload struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return GitHubUser{}, err
	}

	nickname := payload.Name
	if nickname == "" {
		nickname = payload.Login
	}
	return GitHubUser{
		ID:        strconv.FormatInt(payload.ID, 10),
		Nickname:  nickname,
		AvatarURL: payload.AvatarURL,
		Email:     payload.Email,
	}, nil
}

func (c *GitHubClient) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubEmailsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "netshop-frontend-gateway")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch github emails failed: status=%d", resp.StatusCode)
	}

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, email := range emails {
		if email.Primary {
			return email.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}
	return "", nil
}
