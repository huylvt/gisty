# Coding Checklist: Gisty

> Mỗi Task phải hoàn thành và kiểm thử được trước khi chuyển sang Task tiếp theo.

---

## Phase 1: Infrastructure & Project Setup

### Task 1: Docker Infrastructure
**Mục tiêu:** Thiết lập môi trường development với Docker

**Checklist:**
- [ ] Tạo `docker-compose.yml` với các services:
  - [ ] MongoDB (port 27017)
  - [ ] Redis (port 6379)
- [ ] Tạo `.env.example` với các biến môi trường:
  - [ ] Server config (PORT, ENV)
  - [ ] MongoDB config (MONGO_URI, MONGO_DB)
  - [ ] Redis config (REDIS_URI)
  - [ ] AWS S3 config (AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, S3_BUCKET)
- [ ] Tạo `Makefile` với các lệnh cơ bản (up, down, logs)

**Kiểm thử:**
```bash
docker-compose up -d
# MongoDB: mongosh --port 27017
# Redis: redis-cli ping → PONG
```

**Output:** MongoDB và Redis chạy healthy

---

### Task 2: Go Project Structure
**Mục tiêu:** Khởi tạo Go project với cấu trúc chuẩn

**Checklist:**
- [ ] `go mod init github.com/huylvt/gisty`
- [ ] Tạo cấu trúc thư mục:
  ```
  gisty/
  ├── cmd/
  │   └── server/
  │       └── main.go
  ├── internal/
  │   ├── config/
  │   ├── handler/
  │   ├── service/
  │   ├── repository/
  │   └── model/
  ├── pkg/
  │   └── base62/
  └── scripts/
  ```
- [ ] Tạo `internal/config/config.go` - Load config từ env
- [ ] Tạo `cmd/server/main.go` - Entry point cơ bản

**Kiểm thử:**
```bash
go build ./cmd/server
./server --help  # hoặc chạy được không lỗi
```

**Output:** Binary build thành công, chạy được

---

### Task 3: Configuration Management
**Mục tiêu:** Hệ thống config linh hoạt cho nhiều môi trường

**Checklist:**
- [ ] Install dependencies: `github.com/spf13/viper`
- [ ] Tạo `internal/config/config.go`:
  - [ ] Struct Config với các fields: Server, MongoDB, Redis, S3
  - [ ] Hàm `Load()` đọc từ env/file
  - [ ] Validation config required fields
- [ ] Tạo `config.yaml` mẫu cho development

**Kiểm thử:**
```bash
go test ./internal/config/... -v
# Test cases:
# - Load config thành công
# - Báo lỗi khi thiếu required field
```

**Output:** Config load đúng, có validation

---

### Task 4: HTTP Server Foundation
**Mục tiêu:** Setup HTTP server với router và middleware cơ bản

**Checklist:**
- [ ] Install: `github.com/gin-gonic/gin`
- [ ] Tạo `internal/handler/router.go`:
  - [ ] Setup Gin router
  - [ ] Health check endpoint: `GET /health`
  - [ ] Middleware: Logger, Recovery, CORS
- [ ] Tạo `internal/handler/health.go`
- [ ] Integrate vào `cmd/server/main.go`

**Kiểm thử:**
```bash
go run ./cmd/server
curl http://localhost:8080/health
# Response: {"status": "ok", "timestamp": "..."}
```

**Output:** Server chạy, health check trả về 200

---

## Phase 2: Database & Storage Connections

### Task 5: MongoDB Connection
**Mục tiêu:** Kết nối và tương tác với MongoDB

**Checklist:**
- [ ] Install: `go.mongodb.org/mongo-driver/mongo`
- [ ] Tạo `internal/repository/mongodb.go`:
  - [ ] Hàm `NewMongoClient()` - Khởi tạo connection
  - [ ] Hàm `Ping()` - Kiểm tra kết nối
  - [ ] Hàm `Close()` - Đóng connection gracefully
- [ ] Tạo database: `gisty`, collection: `pastes`
- [ ] Integrate vào main.go với graceful shutdown

**Kiểm thử:**
```bash
go run ./cmd/server
# Log: "Connected to MongoDB"
# Shutdown với Ctrl+C: "MongoDB connection closed"
```

**Output:** Kết nối MongoDB thành công, graceful shutdown

---

### Task 6: Redis Connection
**Mục tiêu:** Kết nối Redis cho caching

**Checklist:**
- [ ] Install: `github.com/redis/go-redis/v9`
- [ ] Tạo `internal/repository/redis.go`:
  - [ ] Hàm `NewRedisClient()` - Khởi tạo connection
  - [ ] Hàm `Ping()` - Kiểm tra kết nối
  - [ ] Hàm `Close()` - Đóng connection
- [ ] Integrate vào main.go

**Kiểm thử:**
```bash
go run ./cmd/server
# Log: "Connected to Redis"
curl http://localhost:8080/health
# Response bao gồm redis status
```

**Output:** Kết nối Redis thành công

---

### Task 7: AWS S3 Connection
**Mục tiêu:** Kết nối AWS S3 bucket thực tế

**Checklist:**
- [ ] Install: `github.com/aws/aws-sdk-go-v2` và các sub-packages:
  - [ ] `github.com/aws/aws-sdk-go-v2/config`
  - [ ] `github.com/aws/aws-sdk-go-v2/service/s3`
  - [ ] `github.com/aws/aws-sdk-go-v2/credentials`
- [ ] Tạo `internal/repository/s3.go`:
  - [ ] Hàm `NewS3Client()` - Khởi tạo client từ env credentials
  - [ ] Hàm `HealthCheck()` - Kiểm tra connection (HeadBucket)
  - [ ] Hàm `EnsureBucketExists()` - Verify bucket tồn tại
- [ ] Load credentials từ environment variables
- [ ] Integrate vào main.go

**Kiểm thử:**
```bash
# Đảm bảo .env có đầy đủ AWS credentials
go run ./cmd/server
# Log: "Connected to S3, bucket 'your-bucket-name' verified"
```

**Output:** S3 client kết nối thành công với AWS bucket

---

## Phase 3: Core Services

### Task 8: Base62 Encoding Package
**Mục tiêu:** Utility encode/decode Base62 cho short IDs

**Checklist:**
- [ ] Tạo `pkg/base62/base62.go`:
  - [ ] Charset: `0-9a-zA-Z` (62 ký tự)
  - [ ] Hàm `Encode(num uint64) string`
  - [ ] Hàm `Decode(s string) (uint64, error)`
- [ ] Tạo `pkg/base62/base62_test.go`:
  - [ ] Test encode/decode roundtrip
  - [ ] Test các edge cases (0, max uint64)
  - [ ] Benchmark performance

**Kiểm thử:**
```bash
go test ./pkg/base62/... -v -bench=.
# Tất cả tests pass
# Encode(12345) → "3D7"
# Decode("3D7") → 12345
```

**Output:** Base62 package với 100% test coverage

---

### Task 9: Key Generation Service (KGS)
**Mục tiêu:** Service sinh short_id unique

**Checklist:**
- [ ] Tạo `internal/service/kgs.go`:
  - [ ] Struct `KGS` với MongoDB collection `keys`
  - [ ] Hàm `GenerateKeys(count int)` - Pre-generate keys
  - [ ] Hàm `GetNextKey() (string, error)` - Lấy 1 key unused
  - [ ] Atomic operation: đánh dấu key đã dùng
- [ ] Tạo MongoDB indexes cho performance
- [ ] Tạo `internal/service/kgs_test.go`

**Kiểm thử:**
```bash
go test ./internal/service/... -v -run TestKGS
# Test cases:
# - Generate 100 keys thành công
# - GetNextKey trả về key unique
# - Concurrent GetNextKey không trùng lặp
```

**Output:** KGS hoạt động, đảm bảo unique keys

---

### Task 10: KGS Background Worker
**Mục tiêu:** Auto-replenish keys khi cạn

**Checklist:**
- [ ] Thêm vào `internal/service/kgs.go`:
  - [ ] Hàm `CountUnusedKeys() int64`
  - [ ] Hàm `StartReplenishWorker(ctx context.Context)`
  - [ ] Config: `min_keys_threshold`, `batch_size`
- [ ] Worker chạy mỗi 1 phút, kiểm tra và bổ sung keys
- [ ] Graceful shutdown worker

**Kiểm thử:**
```bash
go run ./cmd/server
# Log: "KGS Worker started"
# Log: "Generated 1000 new keys, total unused: 1000"
# Lấy hết keys → Worker tự động bổ sung
```

**Output:** KGS tự động duy trì pool keys

---

## Phase 4: Storage Service

### Task 11: Content Storage Service
**Mục tiêu:** Service lưu/đọc nội dung từ AWS S3

**Checklist:**
- [ ] Tạo `internal/service/storage.go`:
  - [ ] Hàm `SaveContent(shortID, content string) error`
    - [ ] Nén content với gzip
    - [ ] Upload lên S3 với key: `gisty/{shortID}.gz`
    - [ ] Set ContentEncoding: gzip
  - [ ] Hàm `GetContent(shortID string) (string, error)`
    - [ ] Download từ S3
    - [ ] Decompress gzip
  - [ ] Hàm `DeleteContent(shortID string) error`
- [ ] Set Content-Type: text/plain và metadata phù hợp
- [ ] Handle S3 errors (NoSuchKey, AccessDenied, etc.)

**Kiểm thử:**
```bash
go test ./internal/service/... -v -run TestStorage
# Test cases:
# - Save và Get content khớp nhau
# - Content được nén (size giảm)
# - Delete thành công
# - Get content không tồn tại → error

# Manual test với AWS CLI:
aws s3 ls s3://your-bucket/gisty/
```

**Output:** Storage service hoạt động với AWS S3

---

### Task 12: Cache Service
**Mục tiêu:** Redis cache layer cho hot content

**Checklist:**
- [ ] Tạo `internal/service/cache.go`:
  - [ ] Hàm `Set(shortID, content string, ttl time.Duration) error`
  - [ ] Hàm `Get(shortID string) (string, bool, error)`
  - [ ] Hàm `Delete(shortID string) error`
  - [ ] Hàm `Exists(shortID string) bool`
- [ ] Config: default TTL, max cache size

**Kiểm thử:**
```bash
go test ./internal/service/... -v -run TestCache
# Test cases:
# - Set/Get hoạt động
# - TTL hết hạn → Get trả về not found
# - Delete xóa thành công
```

**Output:** Cache service với TTL support

---

## Phase 5: Data Models & Repository

### Task 13: Paste Model & Repository
**Mục tiêu:** Data model và CRUD operations cho Paste

**Checklist:**
- [ ] Tạo `internal/model/paste.go`:
  ```go
  type Paste struct {
      ShortID        string    `bson:"short_id"`
      UserID         *string   `bson:"user_id,omitempty"`
      ContentKey     string    `bson:"content_key"`
      ExpiresAt      *time.Time `bson:"expires_at,omitempty"`
      CreatedAt      time.Time `bson:"created_at"`
      SyntaxType     string    `bson:"syntax_type"`
      IsPrivate      bool      `bson:"is_private"`
      BurnAfterRead  bool      `bson:"burn_after_read"`
  }
  ```
- [ ] Tạo `internal/repository/paste_repo.go`:
  - [ ] Hàm `Create(paste *Paste) error`
  - [ ] Hàm `GetByShortID(shortID string) (*Paste, error)`
  - [ ] Hàm `Delete(shortID string) error`
  - [ ] Hàm `GetExpired() ([]*Paste, error)`
- [ ] Tạo MongoDB indexes: `short_id` (unique), `expires_at`

**Kiểm thử:**
```bash
go test ./internal/repository/... -v -run TestPasteRepo
# Test cases:
# - Create và GetByShortID
# - Duplicate short_id → error
# - Delete thành công
# - GetExpired trả về đúng records
```

**Output:** Paste repository với full CRUD

---

## Phase 6: API Endpoints

### Task 14: Create Paste API
**Mục tiêu:** `POST /api/v1/pastes` - Tạo paste mới

**Checklist:**
- [ ] Tạo `internal/handler/paste_handler.go`:
  - [ ] Request DTO:
    ```go
    type CreatePasteRequest struct {
        Content    string `json:"content" binding:"required"`
        SyntaxType string `json:"syntax_type"`
        ExpiresIn  string `json:"expires_in"` // "10m", "1h", "1d", "1w", "never", "burn"
        IsPrivate  bool   `json:"is_private"`
    }
    ```
  - [ ] Response DTO:
    ```go
    type CreatePasteResponse struct {
        ShortID   string `json:"short_id"`
        URL       string `json:"url"`
        ExpiresAt string `json:"expires_at,omitempty"`
    }
    ```
- [ ] Tạo `internal/service/paste_service.go`:
  - [ ] Hàm `CreatePaste(req CreatePasteRequest) (*CreatePasteResponse, error)`
  - [ ] Flow: KGS → Storage → MongoDB → Cache (optional)
- [ ] Đăng ký route trong router

**Kiểm thử:**
```bash
curl -X POST http://localhost:8080/api/v1/pastes \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello World", "expires_in": "1h"}'

# Response:
# {
#   "short_id": "xK9a2",
#   "url": "http://localhost:8080/xK9a2",
#   "expires_at": "2026-01-14T15:00:00Z"
# }
```

**Output:** API tạo paste hoạt động end-to-end

---

### Task 15: Get Paste API
**Mục tiêu:** `GET /api/v1/pastes/:id` - Đọc paste

**Checklist:**
- [ ] Thêm handler `GetPaste`:
  - [ ] Response DTO:
    ```go
    type GetPasteResponse struct {
        ShortID    string `json:"short_id"`
        Content    string `json:"content"`
        SyntaxType string `json:"syntax_type"`
        CreatedAt  string `json:"created_at"`
        ExpiresAt  string `json:"expires_at,omitempty"`
    }
    ```
- [ ] Thêm service `GetPaste(shortID string)`:
  - [ ] Check cache first (Cache Hit)
  - [ ] Cache miss → MongoDB → S3 → Update cache
  - [ ] Handle "burn after read": delete sau khi đọc
  - [ ] Handle expired: trả 404
- [ ] Đăng ký route

**Kiểm thử:**
```bash
# Tạo paste trước
curl http://localhost:8080/api/v1/pastes/xK9a2

# Response:
# {
#   "short_id": "xK9a2",
#   "content": "Hello World",
#   "syntax_type": "plaintext",
#   "created_at": "2026-01-14T14:00:00Z"
# }

# Test burn after read:
# Lần 1: 200 OK
# Lần 2: 404 Not Found
```

**Output:** API đọc paste với caching và burn logic

---

### Task 16: Delete Paste API
**Mục tiêu:** `DELETE /api/v1/pastes/:id` - Xóa paste

**Checklist:**
- [ ] Thêm handler `DeletePaste`
- [ ] Thêm service `DeletePaste(shortID string)`:
  - [ ] Xóa từ MongoDB
  - [ ] Xóa từ S3
  - [ ] Xóa từ Cache
- [ ] Đăng ký route
- [ ] (Optional) Yêu cầu auth token để xóa

**Kiểm thử:**
```bash
curl -X DELETE http://localhost:8080/api/v1/pastes/xK9a2
# Response: 204 No Content

curl http://localhost:8080/api/v1/pastes/xK9a2
# Response: 404 Not Found
```

**Output:** API xóa paste hoạt động

---

### Task 17: Short URL Redirect
**Mục tiêu:** `GET /:id` - Redirect/render paste

**Checklist:**
- [ ] Thêm handler cho route `/:id`:
  - [ ] Nếu request Accept: application/json → trả JSON
  - [ ] Nếu request từ browser → render HTML (hoặc redirect)
- [ ] Xử lý content negotiation

**Kiểm thử:**
```bash
# API call
curl -H "Accept: application/json" http://localhost:8080/xK9a2
# → JSON response

# Browser
curl http://localhost:8080/xK9a2
# → HTML hoặc plain text
```

**Output:** Short URL hoạt động cho cả API và browser

---

## Phase 7: Background Workers

### Task 18: Cleanup Worker
**Mục tiêu:** Xóa các paste đã hết hạn

**Checklist:**
- [ ] Tạo `internal/worker/cleanup.go`:
  - [ ] Hàm `StartCleanupWorker(ctx context.Context)`
  - [ ] Chạy mỗi 5 phút
  - [ ] Query MongoDB: `expires_at < now()`
  - [ ] Xóa batch: MongoDB → S3 → Cache
  - [ ] Log số records đã xóa
- [ ] Config: `cleanup_interval`, `batch_size`
- [ ] Integrate vào main.go

**Kiểm thử:**
```bash
# Tạo paste với expires_in: "1m"
# Đợi 2 phút
# Kiểm tra logs: "Cleaned up X expired pastes"
# GET paste → 404
```

**Output:** Cleanup worker xóa paste hết hạn tự động

---

## Phase 8: Security & Rate Limiting

### Task 19: Rate Limiting Middleware
**Mục tiêu:** Giới hạn 5 requests/phút/IP

**Checklist:**
- [ ] Install: `github.com/ulule/limiter/v3`
- [ ] Tạo `internal/middleware/ratelimit.go`:
  - [ ] Rate limit dựa trên IP
  - [ ] Config: `requests_per_minute`
  - [ ] Response 429 khi vượt limit
  - [ ] Header: `X-RateLimit-Remaining`
- [ ] Apply cho POST endpoints

**Kiểm thử:**
```bash
# Gọi POST 6 lần liên tục
for i in {1..6}; do
  curl -X POST http://localhost:8080/api/v1/pastes \
    -d '{"content": "test"}'
done
# Lần thứ 6: 429 Too Many Requests
```

**Output:** Rate limiting hoạt động

---

### Task 20: Input Sanitization
**Mục tiêu:** Chống XSS và injection

**Checklist:**
- [ ] Install: `github.com/microcosm-cc/bluemonday`
- [ ] Tạo `internal/middleware/sanitize.go`:
  - [ ] Sanitize HTML trong content
  - [ ] Validate input length (max 1MB)
  - [ ] Validate syntax_type whitelist
- [ ] Apply trong CreatePaste handler

**Kiểm thử:**
```bash
# Test XSS
curl -X POST http://localhost:8080/api/v1/pastes \
  -d '{"content": "<script>alert(1)</script>"}'
# Content được sanitize hoặc escaped

# Test size limit
curl -X POST http://localhost:8080/api/v1/pastes \
  -d '{"content": "[2MB text]"}'
# Response: 400 Content too large
```

**Output:** Input được validate và sanitize

---

## Phase 9: Syntax Highlighting

### Task 21: Language Detection
**Mục tiêu:** Tự động detect ngôn ngữ lập trình

**Checklist:**
- [ ] Install: `github.com/go-enry/go-enry/v2`
- [ ] Tạo `internal/service/syntax.go`:
  - [ ] Hàm `DetectLanguage(content string) string`
  - [ ] Fallback về "plaintext" nếu không detect được
  - [ ] Map language → syntax highlighter name
- [ ] Integrate vào CreatePaste khi syntax_type rỗng

**Kiểm thử:**
```bash
# Không specify syntax_type
curl -X POST http://localhost:8080/api/v1/pastes \
  -d '{"content": "def hello():\n    print(\"Hi\")"}'

# Response: syntax_type = "python"
```

**Output:** Auto-detect language hoạt động

---

## Phase 10: API Documentation & Testing

### Task 22: OpenAPI/Swagger Documentation
**Mục tiêu:** API documentation tự động

**Checklist:**
- [ ] Install: `github.com/swaggo/swag`
- [ ] Thêm annotations vào handlers
- [ ] Generate docs: `swag init`
- [ ] Serve Swagger UI tại `/docs`

**Kiểm thử:**
```bash
go run ./cmd/server
# Truy cập http://localhost:8080/docs
# → Swagger UI hiển thị đầy đủ endpoints
```

**Output:** API docs có thể truy cập và test

---

### Task 23: Integration Tests
**Mục tiêu:** Test end-to-end flows

**Checklist:**
- [ ] Tạo `tests/integration/`:
  - [ ] `paste_test.go`: Test full CRUD flow
  - [ ] `expiration_test.go`: Test TTL và burn
  - [ ] `ratelimit_test.go`: Test rate limiting
- [ ] Setup test containers (testcontainers-go)
- [ ] CI-ready test commands

**Kiểm thử:**
```bash
go test ./tests/integration/... -v
# Tất cả tests pass
```

**Output:** Integration tests đầy đủ

---

### Task 24: Dockerfile & Production Build
**Mục tiêu:** Container hóa application

**Checklist:**
- [ ] Tạo `Dockerfile`:
  - [ ] Multi-stage build
  - [ ] Scratch/Alpine base image
  - [ ] Non-root user
- [ ] Tạo `docker-compose.prod.yml`
- [ ] Health check trong Dockerfile
- [ ] Build và test image

**Kiểm thử:**
```bash
docker build -t gisty:latest .
docker run -p 8080:8080 gisty:latest
curl http://localhost:8080/health
```

**Output:** Production-ready Docker image

---

## Summary

| Phase | Tasks | Mô tả |
|-------|-------|-------|
| 1 | 1-4 | Infrastructure & Project Setup |
| 2 | 5-7 | Database & Storage Connections |
| 3 | 8-10 | Core Services (Base62, KGS) |
| 4 | 11-12 | Storage & Cache Services |
| 5 | 13 | Data Models & Repository |
| 6 | 14-17 | API Endpoints |
| 7 | 18 | Background Workers |
| 8 | 19-20 | Security & Rate Limiting |
| 9 | 21 | Syntax Highlighting |
| 10 | 22-24 | Documentation & Production |

**Tổng cộng: 24 Tasks**

---

## Progress Tracker

- [ ] Task 1: Docker Infrastructure
- [ ] Task 2: Go Project Structure
- [ ] Task 3: Configuration Management
- [ ] Task 4: HTTP Server Foundation
- [ ] Task 5: MongoDB Connection
- [ ] Task 6: Redis Connection
- [ ] Task 7: S3/MinIO Connection
- [ ] Task 8: Base62 Encoding Package
- [ ] Task 9: Key Generation Service (KGS)
- [ ] Task 10: KGS Background Worker
- [ ] Task 11: Content Storage Service
- [ ] Task 12: Cache Service
- [ ] Task 13: Paste Model & Repository
- [ ] Task 14: Create Paste API
- [ ] Task 15: Get Paste API
- [ ] Task 16: Delete Paste API
- [ ] Task 17: Short URL Redirect
- [ ] Task 18: Cleanup Worker
- [ ] Task 19: Rate Limiting Middleware
- [ ] Task 20: Input Sanitization
- [ ] Task 21: Language Detection
- [ ] Task 22: OpenAPI/Swagger Documentation
- [ ] Task 23: Integration Tests
- [ ] Task 24: Dockerfile & Production Build
