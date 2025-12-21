# Design Patterns

This document describes the design patterns used in this project.

## Dependency Injection

Constructor injection in `main.go` - dependencies flow downward:

```go
// Build order: Config → Clients → Repos → Services → Handlers → Router
cfg := config.Load()
mongoDB := database.NewMongoDB(cfg.MongoURI, cfg.MongoDatabase)
redisCache := cache.NewRedis(cfg.RedisURI)

userRepo := repository.NewUserRepository(mongoDB.Database)
userService := service.NewUserService(userRepo, redisCache)
userHandler := handler.NewUserHandler(userService)

router := router.Setup(userHandler, ...)
```

**Rules:**
- No global state or singletons
- Dependencies passed via constructor
- `defer` for cleanup (e.g., `defer mongoDB.Close()`)

## Router & Middleware Pattern

Route grouping with selective middleware:

```go
v1 := r.Group("/api/v1")
{
    // Public routes (no auth)
    auth := v1.Group("/auth")
    auth.POST("/login", authHandler.Login)

    // Protected routes (with auth middleware)
    users := v1.Group("/users")
    users.Use(middleware.Auth(jwtManager))  // Apply to group
    users.GET("/:id", userHandler.GetUser)
}
```

**Context Passing:**
```go
// Middleware sets value
c.Set("userID", claims.UserID)

// Handler retrieves value
userID, exists := c.Get("userID")
```

## Soft Delete Pattern

Use `deletedAt` timestamp instead of hard delete:

```go
// Model
type VoiceMemo struct {
    // ...
    DeletedAt *time.Time `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
}

// Repository - exclude soft-deleted in queries
filter := bson.M{
    "userId":    userID,
    "deletedAt": bson.M{"$exists": false},  // Only active records
}

// Repository - soft delete
update := bson.M{"$set": bson.M{"deletedAt": time.Now()}}
```

**Benefits:**
- Data recovery possible
- Audit trail preserved
- S3 files retained until hard delete

## Pagination Pattern

Offset-based pagination with metadata:

```go
// Response model
type VoiceMemoListResponse struct {
    Items      []VoiceMemo `json:"items"`
    Pagination Pagination  `json:"pagination"`
}

type Pagination struct {
    Page       int `json:"page"`
    Limit      int `json:"limit"`
    TotalItems int `json:"totalItems"`
    TotalPages int `json:"totalPages"`
}

// Service - calculate pagination
totalPages := total / limit
if total % limit > 0 {
    totalPages++
}

// Repository - apply pagination
skip := (page - 1) * limit
opts := options.Find().
    SetSort(bson.D{{Key: "createdAt", Value: -1}}).  // Newest first
    SetSkip(int64(skip)).
    SetLimit(int64(limit))
```

## Computed Fields Pattern

Fields computed at runtime, not stored:

```go
type VoiceMemo struct {
    AudioFileKey string `json:"-" bson:"audioFileKey"`           // Stored, hidden from JSON
    AudioFileURL string `json:"audioFileUrl" bson:"-"`           // Computed, not stored
}

// Service generates pre-signed URL
for i := range memos {
    url, _ := s.s3Client.GetPresignedURL(ctx, memos[i].AudioFileKey, 1*time.Hour)
    memos[i].AudioFileURL = url
}
```

**Use cases:**
- Pre-signed S3 URLs (expire after 1 hour)
- Computed aggregations
- Derived fields

## Token Pattern (JWT + Refresh)

Short-lived JWT access token + long-lived opaque refresh token:

```
┌─────────────────┐     ┌─────────────────┐
│  Access Token   │     │  Refresh Token  │
├─────────────────┤     ├─────────────────┤
│ Type: JWT       │     │ Type: Opaque    │
│ TTL: 15 min     │     │ TTL: 7 days     │
│ Contains: userID│     │ Prefix: rf_     │
│ Stored: Client  │     │ Stored: DB+Redis│
└─────────────────┘     └─────────────────┘
```

```go
// Generate opaque refresh token
func generateRefreshToken() (string, error) {
    bytes := make([]byte, 32)
    rand.Read(bytes)
    return "rf_" + hex.EncodeToString(bytes), nil
}

// Refresh flow: Cache first, then DB
userID, _ := s.cache.GetRefreshToken(ctx, token)
if userID == "" {
    refreshToken, _ := s.refreshTokenRepo.FindByToken(ctx, token)
    userID = refreshToken.UserID.Hex()
    // Cache for next time
    s.cache.SetRefreshToken(ctx, token, userID, ttl)
}
```

## Configuration Pattern

Required vs optional environment variables:

```go
type Config struct {
    MongoURI    string        // Required - fatal if missing
    RedisURI    string        // Optional - has default
    S3UseSSL    bool          // Boolean from string
    TokenExpiry time.Duration // Duration parsing
}

func Load() *Config {
    _ = godotenv.Load()  // Ignore error - env vars may be set directly
    return &Config{
        MongoURI:    getEnvRequired("MONGO_URI"),     // Panics if missing
        RedisURI:    getEnv("REDIS_URI", "localhost:6379"),
        S3UseSSL:    getEnv("S3_USE_SSL", "false") == "true",
        TokenExpiry: parseDuration(getEnv("TOKEN_EXPIRY", "15m")),
    }
}
```

## Cache Key Pattern

Namespaced keys with consistent format:

```go
// Key generators
func UserCacheKey(userID string) string {
    return fmt.Sprintf("user:%s", userID)
}

func RefreshTokenCacheKey(token string) string {
    return fmt.Sprintf("refresh:%s", token)
}

// Results in: "user:507f1f77bcf86cd799439011"
//             "refresh:rf_8a7b3c9d..."
```

## Interface-Based Repository

Interfaces for testability and abstraction:

```go
// Interface in repository package
type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error)
    FindByEmail(ctx context.Context, email string) (*models.User, error)
    // ...
}

// Implementation is unexported
type userRepository struct {
    collection *mongo.Collection
}

// Constructor returns interface
func NewUserRepository(db *mongo.Database) UserRepository {
    return &userRepository{collection: db.Collection("users")}
}

// Service depends on interface (mockable)
type UserService struct {
    repo  UserRepository  // Interface, not concrete type
    cache *cache.Redis
}
```

## Background Job Processing Pattern

Worker pool with in-memory queue for async tasks:

```go
// Queue with bounded capacity
type MemoryQueue struct {
    jobs     chan TranscriptionJob
    capacity int
    mu       sync.RWMutex
    closed   bool
}

// Processor with multiple workers
type Processor struct {
    queue       *MemoryQueue
    transcriber transcription.Service
    updater     TranscriptionUpdater
    workerCount int
    wg          sync.WaitGroup
    shutdownCh  chan struct{}
}

// Start workers
func (p *Processor) Start(ctx context.Context) {
    for i := 0; i < p.workerCount; i++ {
        p.wg.Add(1)
        go p.worker(ctx, i)
    }
}

// Graceful stop - wait for workers
func (p *Processor) Stop() {
    close(p.shutdownCh)
    p.queue.Close()
    p.wg.Wait()
}
```

**Features:**
- Bounded queue prevents memory exhaustion
- Multiple workers for parallelism
- Exponential backoff retry (5s, 10s, 20s)
- Max retries before marking as failed
- Graceful shutdown waits for in-flight jobs

## Graceful Shutdown Pattern

Proper shutdown sequence for HTTP server and background workers:

```go
// Create HTTP server (not r.Run())
srv := &http.Server{
    Addr:    ":8080",
    Handler: router,
}

// Start in goroutine
go func() {
    if err := srv.ListenAndServe(); err != http.ErrServerClosed {
        log.Fatal(err)
    }
}()

// Wait for signal
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
<-sigCh

// Shutdown sequence
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// 1. Stop accepting new HTTP requests, drain existing
srv.Shutdown(ctx)

// 2. Cancel context to signal workers
cancel()

// 3. Stop background processor (waits for workers)
processor.Stop()
```

**Order matters:**
1. HTTP server first (drain connections)
2. Cancel context (signal workers)
3. Stop processor (wait for completion)

## Status-Driven Workflow Pattern

State machine for multi-step async processes:

```go
type VoiceMemoStatus string

const (
    StatusPendingUpload VoiceMemoStatus = "pending_upload"
    StatusTranscribing  VoiceMemoStatus = "transcribing"
    StatusReady         VoiceMemoStatus = "ready"
    StatusFailed        VoiceMemoStatus = "failed"
)
```

**State transitions:**
```
pending_upload → transcribing → ready
                      ↓
                   failed → transcribing (retry)
```

**Atomic status updates with conditions:**
```go
// Only update if in expected state (prevents race conditions)
func UpdateStatusConditional(ctx, id, fromStatus, toStatus) error {
    filter := bson.M{
        "_id":    id,
        "status": fromStatus,  // Condition
    }
    update := bson.M{"$set": bson.M{"status": toStatus}}
    // Silently succeeds if status already changed
    _, err := collection.UpdateOne(ctx, filter, update)
    return err
}
```

**Benefits:**
- Clear visibility into process state
- Safe concurrent operations
- Idempotent retries
- Easy failure recovery
