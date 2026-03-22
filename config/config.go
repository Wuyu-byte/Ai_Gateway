package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	App       AppConfig
	MySQL     MySQLConfig
	Redis     RedisConfig
	Auth      AuthConfig
	RateLimit RateLimitConfig
	Scheduler SchedulerConfig
	Logging   LoggingConfig
	Providers ProviderGroupConfig
}

type AppConfig struct {
	Port int
	Env  string
}

type MySQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	Charset  string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type AuthConfig struct {
	JWTSecret        string
	JWTExpireMinutes int
}

type RateLimitConfig struct {
	PerMinute int
}

type SchedulerConfig struct {
	HealthCheckIntervalSec int
	HealthCheckTimeoutSec  int
}

type LoggingConfig struct {
	QueueSize       int
	WorkerCount     int
	BatchSize       int
	FlushIntervalMS int
}

type ProviderGroupConfig struct {
	OpenAI   ProviderConfig
	DeepSeek ProviderConfig
	Claude   ProviderConfig
}

type ProviderConfig struct {
	BaseURL string
	APIKeys []string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		App: AppConfig{
			Port: getEnvAsInt("APP_PORT", 8080),
			Env:  getEnv("APP_ENV", "dev"),
		},
		MySQL: MySQLConfig{
			Host:     getEnv("MYSQL_HOST", "127.0.0.1"),
			Port:     getEnvAsInt("MYSQL_PORT", 3306),
			User:     getEnv("MYSQL_USER", "root"),
			Password: getEnv("MYSQL_PASSWORD", ""),
			DBName:   getEnv("MYSQL_DBNAME", "ai_gateway"),
			Charset:  getEnv("MYSQL_CHARSET", "utf8mb4"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Auth: AuthConfig{
			JWTSecret:        getEnv("JWT_SECRET", "change-me-in-production"),
			JWTExpireMinutes: getEnvAsInt("JWT_EXPIRE_MINUTES", 30),
		},
		RateLimit: RateLimitConfig{
			PerMinute: getEnvAsInt("RATE_LIMIT_PER_MINUTE", 60),
		},
		Scheduler: SchedulerConfig{
			HealthCheckIntervalSec: getEnvAsInt("SCHEDULER_HEALTHCHECK_INTERVAL_SEC", 10),
			HealthCheckTimeoutSec:  getEnvAsInt("SCHEDULER_HEALTHCHECK_TIMEOUT_SEC", 5),
		},
		Logging: LoggingConfig{
			QueueSize:       getEnvAsInt("ASYNC_LOG_QUEUE_SIZE", 2048),
			WorkerCount:     getEnvAsInt("ASYNC_LOG_WORKER_COUNT", 4),
			BatchSize:       getEnvAsInt("ASYNC_LOG_BATCH_SIZE", 50),
			FlushIntervalMS: getEnvAsInt("ASYNC_LOG_FLUSH_INTERVAL_MS", 1000),
		},
		Providers: ProviderGroupConfig{
			OpenAI: ProviderConfig{
				BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
				APIKeys: getEnvAsSlice("OPENAI_KEYS"),
			},
			DeepSeek: ProviderConfig{
				BaseURL: getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1"),
				APIKeys: getEnvAsSlice("DEEPSEEK_KEYS"),
			},
			Claude: ProviderConfig{
				BaseURL: getEnv("CLAUDE_BASE_URL", "https://api.anthropic.com/v1"),
				APIKeys: getEnvAsSlice("CLAUDE_KEYS"),
			},
		},
	}
}

func (c MySQLConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.Charset,
	)
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func getEnvAsInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return number
}

func getEnvAsSlice(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}
