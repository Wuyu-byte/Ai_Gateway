# AI Gateway v3

AI Gateway v3 is a production-style LLM API gateway built with Go, Gin, MySQL, and Redis. It supports provider aggregation, SSE streaming, provider scheduling, async logging, Prometheus metrics, and a dual-layer auth model:

- JWT for user and API key management
- API Key for AI traffic

## Architecture

```text
User
  -> /auth/register
  -> /auth/login
  -> JWT
  -> /apikey/create | /apikey/list | /apikey/:id
  -> API Key
  -> /v1/chat/completions

Gateway
  -> JWT middleware for management APIs
  -> API Key middleware for AI APIs
  -> Redis rate limit
  -> Scheduler
  -> Provider
  -> Async usage log workers
  -> MySQL
```

## Project Structure

```text
ai-gateway
├── api
│   ├── apikey_handler.go
│   ├── auth_handler.go
│   ├── chat_handler.go
│   └── stats_handler.go
├── cmd
│   └── main.go
├── config
│   └── config.go
├── logger
│   └── async_logger.go
├── metrics
│   └── collector.go
├── middleware
│   ├── apikey.go
│   ├── jwt.go
│   └── ratelimit.go
├── model
│   ├── apikey.go
│   ├── usage.go
│   └── user.go
├── pkg
│   └── redis.go
├── provider
│   ├── claude.go
│   ├── deepseek.go
│   ├── openai.go
│   ├── openai_compatible.go
│   └── provider.go
├── repository
│   ├── apikey_repo.go
│   ├── usage_repo.go
│   └── user_repo.go
├── router
│   └── router.go
├── scheduler
│   └── scheduler.go
├── service
│   ├── apikey_service.go
│   ├── auth_service.go
│   ├── chat_service.go
│   ├── stats_service.go
│   └── usage_stats_service.go
├── sql
│   └── init.sql
├── .env.example
├── docker-compose.yml
├── go.mod
└── README.md
```

## Environment Variables

Important variables:

```env
APP_PORT=8080

MYSQL_HOST=119.3.255.75
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=xxxx
MYSQL_DBNAME=ai_gateway

REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0

JWT_SECRET=change-me-in-production
JWT_EXPIRE_MINUTES=30

RATE_LIMIT_PER_MINUTE=60

OPENAI_KEYS=sk-openai-key-1
DEEPSEEK_KEYS=sk-deepseek-key-1
CLAUDE_KEYS=sk-claude-key-1
```

## Start

```powershell
docker run -d -p 6379:6379 --name ai-gateway-redis redis
```

```powershell
go mod tidy
go run cmd/main.go
```

## API Flow

### 1. Register

```powershell
$registerBody = @{
  username = "demo_user"
  password = "Demo@123456"
} | ConvertTo-Json

Invoke-RestMethod `
  -Uri "http://localhost:8080/auth/register" `
  -Method Post `
  -ContentType "application/json" `
  -Body $registerBody
```

### 2. Login and get JWT

```powershell
$loginBody = @{
  username = "demo_user"
  password = "Demo@123456"
} | ConvertTo-Json

$loginResp = Invoke-RestMethod `
  -Uri "http://localhost:8080/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body $loginBody

$jwt = $loginResp.access_token
```

### 3. Create API Key with JWT

```powershell
$createKeyBody = @{
  name = "default-key"
  rate_limit = 60
} | ConvertTo-Json

$keyResp = Invoke-RestMethod `
  -Uri "http://localhost:8080/apikey/create" `
  -Method Post `
  -Headers @{ Authorization = "Bearer $jwt" } `
  -ContentType "application/json" `
  -Body $createKeyBody

$apiKey = $keyResp.data.key
```

### 4. Call AI API with API Key

```powershell
$chatBody = @{
  model = "deepseek-chat"
  messages = @(
    @{
      role = "user"
      content = "Please introduce yourself in one sentence."
    }
  )
} | ConvertTo-Json -Depth 5

Invoke-RestMethod `
  -Uri "http://localhost:8080/v1/chat/completions" `
  -Method Post `
  -Headers @{ Authorization = "Bearer $apiKey" } `
  -ContentType "application/json" `
  -Body $chatBody
```

## Main Endpoints

- `POST /auth/register`
- `POST /auth/login`
- `POST /apikey/create`
- `GET /apikey/list`
- `DELETE /apikey/:id`
- `POST /v1/chat/completions`
- `GET /v1/stats/usage/daily`
- `GET /v1/stats/usage/users`
- `GET /v1/stats/usage/providers`
- `GET /v1/providers/status`
- `GET /metrics`

## Notes

- JWT is only for management APIs.
- `/v1/chat/completions` only accepts API Key.
- API Keys are stored as hashes in MySQL, not plaintext.
- Redis rate limiting is isolated by API key identity.
- Streaming is supported for OpenAI-compatible providers.
