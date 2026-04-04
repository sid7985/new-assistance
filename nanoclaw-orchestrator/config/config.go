package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	MiniMaxAPIKey       string
	MiniMaxGroupID      string
	PerplexityAPIKey    string
	ProjectDir          string
	WaitMinutes         time.Duration
	RetryMinutes        time.Duration
	PerplexitySearchX   int
	PerplexitySearchY   int
	PerplexityResponseX int
	PerplexityResponseY int
	TelegramBotToken    string
	TelegramChatID      string
	VeniceAPIKey        string
	VeniceModel         string
	MachineName         string
}

func Load() *Config {
	// Try loading .env from current dir and parent dir
	if err := godotenv.Load(); err != nil {
		if err2 := godotenv.Load("../.env"); err2 != nil {
			fmt.Println("⚠️  No .env file found. Set environment variables manually or create a .env file.")
		}
	}
	return &Config{
		MiniMaxAPIKey:       getEnv("MINIMAX_API_KEY", ""),
		MiniMaxGroupID:      getEnv("MINIMAX_GROUP_ID", ""),
		PerplexityAPIKey:    getEnv("PERPLEXITY_API_KEY", ""),
		ProjectDir:          getEnv("PROJECT_DIR", "."),
		WaitMinutes:         getEnvDuration("WAIT_MINUTES", 1),
		RetryMinutes:        getEnvDuration("RETRY_MINUTES", 3),
		PerplexitySearchX:   getEnvInt("PPLX_SEARCH_X", 500),
		PerplexitySearchY:   getEnvInt("PPLX_SEARCH_Y", 300),
		PerplexityResponseX: getEnvInt("PPLX_RESPONSE_X", 800),
		PerplexityResponseY: getEnvInt("PPLX_RESPONSE_Y", 600),
		TelegramBotToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:      getEnv("TELEGRAM_CHAT_ID", ""),
		VeniceAPIKey:        getEnv("VENICE_API_KEY", ""),
		VeniceModel:         getEnv("VENICE_MODEL", "venice-m1-pro"), // Default Mithril-class model on Venice
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue int) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return time.Duration(parsed) * time.Minute
		}
	}
	return time.Duration(defaultValue) * time.Minute
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
