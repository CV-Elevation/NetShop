package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr string

	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackURL  string

	JWTSecret    string
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
	CookieSecure bool

	UserServiceAddr        string
	EmailServiceAddr       string
	ProductServiceAddr     string
	AdServiceAddr          string
	RecommendServiceAddr   string
	CartServiceAddr        string
	AIAssistantServiceAddr string
}

func Load() Config {
	loadDotEnv()

	accessMinutes := getEnvInt("ACCESS_TTL_MINUTES", 15)
	refreshHours := getEnvInt("REFRESH_TTL_HOURS", 24*7)

	return Config{
		//监听端口
		ListenAddr: getEnv("PORT", ":8080"),
		//Github Oauth相关的组件
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GitHubCallbackURL:  getEnv("GITHUB_CALLBACK_URL", "http://localhost:8080/auth/github/callback"),
		//生成JWT的相关配置
		JWTSecret:  getEnv("JWT_SECRET", "dev-secret-change-me"),
		AccessTTL:  time.Duration(accessMinutes) * time.Minute,
		RefreshTTL: time.Duration(refreshHours) * time.Hour,
		//表示是否强制在https环境下发送
		CookieSecure: getEnvBool("COOKIE_SECURE", false),
		//后方服务grpc地址
		UserServiceAddr:        getEnv("USER_SERVICE_ADDR", "localhost:50051"),
		EmailServiceAddr:       getEnv("EMAIL_SERVICE_ADDR", "localhost:50052"),
		ProductServiceAddr:     getEnv("PRODUCT_SERVICE_ADDR", "localhost:50053"),
		AdServiceAddr:          getEnv("AD_SERVICE_ADDR", "localhost:50055"),
		RecommendServiceAddr:   getEnv("RECOMMEND_SERVICE_ADDR", "localhost:50054"),
		CartServiceAddr:        getEnv("CART_SERVICE_ADDR", "localhost:50056"),
		AIAssistantServiceAddr: getEnv("AIASSISTANT_SERVICE_ADDR", "localhost:50057"),
	}
}

func loadDotEnv() {
	if err := loadEnvFile(".env"); err == nil {
		return
	}
	_ = loadEnvFile(filepath.Join("services", "frontend", ".env"))
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, strings.Trim(value, `"'`))
	}

	return scanner.Err()
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
