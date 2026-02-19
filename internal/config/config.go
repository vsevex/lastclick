package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env            string
	HTTPAddr       string
	BotToken       string
	MiniAppURL     string
	DatabaseURL    string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	WSReadLimit    int64
	WSPingInterval time.Duration
}

func Load() (*Config, error) {
	env := getenv("ENV", "development")

	// Load .env.{ENV} first, then .env as fallback
	loadEnvFile(".env." + env)
	loadEnvFile(".env")

	cfg := &Config{
		Env:            env,
		HTTPAddr:       getenv("HTTP_ADDR", ":8080"),
		BotToken:       getenv("BOT_TOKEN", ""),
		MiniAppURL:     getenv("MINI_APP_URL", "https://lastclick.app"),
		DatabaseURL:    getenv("DATABASE_URL", "postgres://lastclick:lastclick@localhost:5432/lastclick?sslmode=disable"),
		RedisAddr:      getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getenv("REDIS_PASSWORD", ""),
		RedisDB:        getenvInt("REDIS_DB", 0),
		WSReadLimit:    int64(getenvInt("WS_READ_LIMIT", 4096)),
		WSPingInterval: time.Duration(getenvInt("WS_PING_INTERVAL_SEC", 30)) * time.Second,
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	return cfg, nil
}

// loadEnvFile parses a KEY=VALUE file and sets any keys not already present in os env.
func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		// Don't override existing env vars
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
