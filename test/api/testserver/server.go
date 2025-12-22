//go:build api

// Package testserver provides a fully wired test server for API integration tests.
package testserver

import (
	"context"
	"time"

	"gin-sample/internal/authz"
	"gin-sample/internal/cache"
	"gin-sample/internal/handler"
	"gin-sample/internal/queue"
	"gin-sample/internal/repository"
	"gin-sample/internal/router"
	"gin-sample/internal/service"
	"gin-sample/internal/storage"
	"gin-sample/internal/transcription"
	"gin-sample/pkg/auth"
	"gin-sample/test/api/testdb"

	"github.com/gin-gonic/gin"
)

const (
	// TestAccessTokenSecret is the JWT secret used in tests.
	TestAccessTokenSecret = "test-secret-key-for-api-tests"
	// TestAccessTokenExpiry is the access token expiry time used in tests.
	TestAccessTokenExpiry = 15 * time.Minute
	// TestRefreshTokenExpiry is the refresh token expiry time used in tests.
	TestRefreshTokenExpiry = 7 * 24 * time.Hour
	// TestDBName is the database name used in tests.
	TestDBName = "test_api"
)

// TestServer holds all dependencies for API integration tests.
type TestServer struct {
	// Router is the Gin engine for making HTTP requests.
	Router *gin.Engine

	// Containers
	MongoDB *testdb.MongoContainer
	Redis   *testdb.RedisContainer
	MinIO   *testdb.MinIOContainer

	// Repositories (for direct database access in tests)
	UserRepo           repository.UserRepository
	RefreshTokenRepo   repository.RefreshTokenRepository
	VoiceMemoRepo      repository.VoiceMemoRepository
	TeamRepo           repository.TeamRepository
	TeamMemberRepo     repository.TeamMemberRepository
	TeamInvitationRepo repository.TeamInvitationRepository

	// Services (for direct service access in tests)
	AuthService           service.AuthServicer
	UserService           service.UserServicer
	VoiceMemoService      service.VoiceMemoServicer
	TeamService           service.TeamServicer
	TeamMemberService     service.TeamMemberServicer
	TeamInvitationService service.TeamInvitationServicer

	// Auth
	JWTManager *auth.JWTManager

	// Queue
	TranscriptionQueue     *queue.MemoryQueue
	TranscriptionProcessor *queue.Processor
	transcriptionService   transcription.Service
}

// New creates a new test server with all dependencies wired up.
func New(ctx context.Context) (*TestServer, error) {
	gin.SetMode(gin.TestMode)

	// Start containers
	mongoDB, err := testdb.SetupMongoDB(ctx, TestDBName)
	if err != nil {
		return nil, err
	}

	redisContainer, err := testdb.SetupRedis(ctx)
	if err != nil {
		_ = mongoDB.Cleanup(ctx)
		return nil, err
	}

	minioContainer, err := testdb.SetupMinIO(ctx)
	if err != nil {
		_ = mongoDB.Cleanup(ctx)
		_ = redisContainer.Cleanup(ctx)
		return nil, err
	}

	// Create cache (uses real Redis)
	redisCache := cache.NewRedis(redisContainer.URI)

	// Create storage (uses real MinIO)
	s3Client := storage.NewS3Client(
		minioContainer.Endpoint,
		minioContainer.AccessKey,
		minioContainer.SecretKey,
		minioContainer.Bucket,
		false, // useSSL
	)

	// JWT Manager
	jwtManager := auth.NewJWTManager(TestAccessTokenSecret, TestAccessTokenExpiry)

	// Repository layer
	userRepo := repository.NewUserRepository(mongoDB.Database)
	refreshTokenRepo := repository.NewRefreshTokenRepository(mongoDB.Database)
	voiceMemoRepo := repository.NewVoiceMemoRepository(mongoDB.Database)
	teamRepo := repository.NewTeamRepository(mongoDB.Database)
	teamMemberRepo := repository.NewTeamMemberRepository(mongoDB.Database)
	teamInvitationRepo := repository.NewTeamInvitationRepository(mongoDB.Database)

	// Authorization
	authorizer := authz.NewLocalAuthorizer(teamMemberRepo)

	// Transcription queue and processor
	transcriptionQueue := queue.NewMemoryQueue(100)
	transcriptionService := transcription.NewMockService()

	// Service layer
	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		redisCache,
		jwtManager,
		TestAccessTokenExpiry,
		TestRefreshTokenExpiry,
	)
	userService := service.NewUserService(userRepo, redisCache)
	voiceMemoService := service.NewVoiceMemoService(voiceMemoRepo, s3Client, transcriptionQueue)
	teamService := service.NewTeamService(teamRepo, teamMemberRepo, teamInvitationRepo, voiceMemoRepo)
	teamMemberService := service.NewTeamMemberService(teamMemberRepo, userRepo, teamRepo)
	teamInvitationService := service.NewTeamInvitationService(teamInvitationRepo, teamMemberRepo, teamRepo, userRepo)

	// Transcription processor
	transcriptionProcessor := queue.NewProcessor(transcriptionQueue, transcriptionService, voiceMemoRepo, 2)

	// Handler layer
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	voiceMemoHandler := handler.NewVoiceMemoHandler(voiceMemoService)
	teamHandler := handler.NewTeamHandler(teamService)
	teamMemberHandler := handler.NewTeamMemberHandler(teamMemberService)
	invitationHandler := handler.NewTeamInvitationHandler(teamInvitationService, userService)

	// Router
	r := router.Setup(&router.Config{
		AuthHandler:       authHandler,
		UserHandler:       userHandler,
		VoiceMemoHandler:  voiceMemoHandler,
		TeamHandler:       teamHandler,
		TeamMemberHandler: teamMemberHandler,
		InvitationHandler: invitationHandler,
		JWTManager:        jwtManager,
		Authorizer:        authorizer,
	})

	return &TestServer{
		Router:                 r,
		MongoDB:                mongoDB,
		Redis:                  redisContainer,
		MinIO:                  minioContainer,
		UserRepo:               userRepo,
		RefreshTokenRepo:       refreshTokenRepo,
		VoiceMemoRepo:          voiceMemoRepo,
		TeamRepo:               teamRepo,
		TeamMemberRepo:         teamMemberRepo,
		TeamInvitationRepo:     teamInvitationRepo,
		AuthService:            authService,
		UserService:            userService,
		VoiceMemoService:       voiceMemoService,
		TeamService:            teamService,
		TeamMemberService:      teamMemberService,
		TeamInvitationService:  teamInvitationService,
		JWTManager:             jwtManager,
		TranscriptionQueue:     transcriptionQueue,
		TranscriptionProcessor: transcriptionProcessor,
		transcriptionService:   transcriptionService,
	}, nil
}

// Cleanup terminates all containers.
func (ts *TestServer) Cleanup(ctx context.Context) {
	if ts.MinIO != nil {
		_ = ts.MinIO.Cleanup(ctx)
	}
	if ts.Redis != nil {
		_ = ts.Redis.Cleanup(ctx)
	}
	if ts.MongoDB != nil {
		_ = ts.MongoDB.Cleanup(ctx)
	}
}

// StartTranscriptionProcessor starts the transcription processor.
func (ts *TestServer) StartTranscriptionProcessor(ctx context.Context) {
	ts.TranscriptionProcessor.Start(ctx)
}

// StopTranscriptionProcessor stops the transcription processor and resets the queue.
// This ensures the queue can be used by subsequent tests.
func (ts *TestServer) StopTranscriptionProcessor() {
	ts.TranscriptionProcessor.Stop()
	// Reset the queue so it can be used again
	ts.TranscriptionQueue.Reset()
	// Create a new processor since the old one has shutdown state
	ts.TranscriptionProcessor = queue.NewProcessor(ts.TranscriptionQueue, ts.transcriptionService, ts.VoiceMemoRepo, 2)
}
