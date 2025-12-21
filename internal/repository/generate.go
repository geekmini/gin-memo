package repository

//go:generate mockgen -destination=mocks/mock_repositories.go -package=mocks gin-sample/internal/repository UserRepository,RefreshTokenRepository,TeamRepository,TeamMemberRepository,TeamInvitationRepository,VoiceMemoRepository
