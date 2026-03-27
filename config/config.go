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
			Port: getEnvAsInt("APP_PORT"),
			Env:  getEnv("APP_ENV"),
		},
		MySQL: MySQLConfig{
			Host:     getEnv("MYSQL_HOST"),
			Port:     getEnvAsInt("MYSQL_PORT"),
			User:     getEnv("MYSQL_USER"),
			Password: getEnv("MYSQL_PASSWORD"),
			DBName:   getEnv("MYSQL_DBNAME"),
			Charset:  getEnv("MYSQL_CHARSET"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR"),
			Password: getEnv("REDIS_PASSWORD"),
			DB:       getEnvAsInt("REDIS_DB"),
		},
		Auth: AuthConfig{
			JWTSecret:        getEnv("JWT_SECRET"),
			JWTExpireMinutes: getEnvAsInt("JWT_EXPIRE_MINUTES"),
		},
		RateLimit: RateLimitConfig{
			PerMinute: getEnvAsInt("RATE_LIMIT_PER_MINUTE"),
		},
		Scheduler: SchedulerConfig{
			HealthCheckIntervalSec: getEnvAsInt("SCHEDULER_HEALTHCHECK_INTERVAL_SEC"),
			HealthCheckTimeoutSec:  getEnvAsInt("SCHEDULER_HEALTHCHECK_TIMEOUT_SEC"),
		},
		Logging: LoggingConfig{
			QueueSize:       getEnvAsInt("ASYNC_LOG_QUEUE_SIZE"),
			WorkerCount:     getEnvAsInt("ASYNC_LOG_WORKER_COUNT"),
			BatchSize:       getEnvAsInt("ASYNC_LOG_BATCH_SIZE"),
			FlushIntervalMS: getEnvAsInt("ASYNC_LOG_FLUSH_INTERVAL_MS"),
		},
		Providers: ProviderGroupConfig{
			OpenAI: ProviderConfig{
				BaseURL: getEnv("OPENAI_BASE_URL"),
				APIKeys: getEnvAsSlice("OPENAI_KEYS"),
			},
			DeepSeek: ProviderConfig{
				BaseURL: getEnv("DEEPSEEK_BASE_URL"),
				APIKeys: getEnvAsSlice("DEEPSEEK_KEYS"),
			},
			Claude: ProviderConfig{
				BaseURL: getEnv("CLAUDE_BASE_URL"),
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

func getEnv(key string) string {
	raw, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("missing required env: %s", key))
	}

	return strings.TrimSpace(raw)
}

func getEnvAsInt(key string) int {
	raw, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("missing required env: %s", key))
	}

	number, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		panic(fmt.Sprintf("invalid int env %s=%q", key, raw))
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
