package repository

import (
	"context"
	"testing"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewUserRepository(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.Database)

	assert.NotNil(t, repo)
}

func TestUserRepository_Create(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.Database)
	ctx := context.Background()

	t.Run("successfully creates user", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		user := &models.User{
			Email:    "test@example.com",
			Password: "hashedpassword",
			Name:     "Test User",
		}

		err := repo.Create(ctx, user)

		require.NoError(t, err)
		assert.False(t, user.ID.IsZero())
		assert.NotZero(t, user.CreatedAt)
		assert.NotZero(t, user.UpdatedAt)
	})

	t.Run("returns error for duplicate email", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		user1 := &models.User{
			Email:    "duplicate@example.com",
			Password: "hashedpassword",
			Name:     "User 1",
		}
		err := repo.Create(ctx, user1)
		require.NoError(t, err)

		user2 := &models.User{
			Email:    "duplicate@example.com",
			Password: "hashedpassword",
			Name:     "User 2",
		}
		err = repo.Create(ctx, user2)

		assert.Equal(t, apperrors.ErrUserAlreadyExists, err)
	})
}

func TestUserRepository_FindByID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.Database)
	ctx := context.Background()

	t.Run("finds existing user", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		user := &models.User{
			Email:    "findbyid@example.com",
			Password: "hashedpassword",
			Name:     "Find By ID User",
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, user.ID)

		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, user.Name, found.Name)
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		nonExistentID := primitive.NewObjectID()
		found, err := repo.FindByID(ctx, nonExistentID)

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})
}

func TestUserRepository_FindByEmail(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.Database)
	ctx := context.Background()

	t.Run("finds user by email", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		user := &models.User{
			Email:    "findbyemail@example.com",
			Password: "hashedpassword",
			Name:     "Find By Email User",
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		found, err := repo.FindByEmail(ctx, "findbyemail@example.com")

		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Email, found.Email)
	})

	t.Run("returns error for non-existent email", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		found, err := repo.FindByEmail(ctx, "nonexistent@example.com")

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})
}

func TestUserRepository_FindAll(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.Database)
	ctx := context.Background()

	t.Run("returns all users", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		user1 := &models.User{Email: "user1@example.com", Password: "pass", Name: "User 1"}
		user2 := &models.User{Email: "user2@example.com", Password: "pass", Name: "User 2"}
		user3 := &models.User{Email: "user3@example.com", Password: "pass", Name: "User 3"}

		require.NoError(t, repo.Create(ctx, user1))
		require.NoError(t, repo.Create(ctx, user2))
		require.NoError(t, repo.Create(ctx, user3))

		users, err := repo.FindAll(ctx)

		require.NoError(t, err)
		assert.Len(t, users, 3)
	})

	t.Run("returns empty slice when no users", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		users, err := repo.FindAll(ctx)

		require.NoError(t, err)
		assert.NotNil(t, users)
		assert.Len(t, users, 0)
	})
}

func TestUserRepository_Update(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.Database)
	ctx := context.Background()

	t.Run("updates user name", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		user := &models.User{
			Email:    "update@example.com",
			Password: "hashedpassword",
			Name:     "Original Name",
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		newName := "Updated Name"
		updated, err := repo.Update(ctx, user.ID, &models.UpdateUserRequest{Name: &newName})

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
	})

	t.Run("updates user email", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		user := &models.User{
			Email:    "original@example.com",
			Password: "hashedpassword",
			Name:     "Test User",
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		newEmail := "updated@example.com"
		updated, err := repo.Update(ctx, user.ID, &models.UpdateUserRequest{Email: &newEmail})

		require.NoError(t, err)
		assert.Equal(t, "updated@example.com", updated.Email)
	})

	t.Run("returns error when updating to existing email", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		user1 := &models.User{Email: "user1@example.com", Password: "pass", Name: "User 1"}
		user2 := &models.User{Email: "user2@example.com", Password: "pass", Name: "User 2"}
		require.NoError(t, repo.Create(ctx, user1))
		require.NoError(t, repo.Create(ctx, user2))

		existingEmail := "user1@example.com"
		_, err := repo.Update(ctx, user2.ID, &models.UpdateUserRequest{Email: &existingEmail})

		assert.Equal(t, apperrors.ErrUserAlreadyExists, err)
	})

	t.Run("allows user to keep same email", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		user := &models.User{
			Email:    "keepemail@example.com",
			Password: "hashedpassword",
			Name:     "Test User",
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		sameEmail := "keepemail@example.com"
		newName := "New Name"
		updated, err := repo.Update(ctx, user.ID, &models.UpdateUserRequest{
			Email: &sameEmail,
			Name:  &newName,
		})

		require.NoError(t, err)
		assert.Equal(t, "keepemail@example.com", updated.Email)
		assert.Equal(t, "New Name", updated.Name)
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		nonExistentID := primitive.NewObjectID()
		newName := "New Name"
		_, err := repo.Update(ctx, nonExistentID, &models.UpdateUserRequest{Name: &newName})

		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})
}

func TestUserRepository_Delete(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.Database)
	ctx := context.Background()

	t.Run("deletes existing user", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		user := &models.User{
			Email:    "delete@example.com",
			Password: "hashedpassword",
			Name:     "Delete Me",
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		err = repo.Delete(ctx, user.ID)

		require.NoError(t, err)

		// Verify user is deleted
		_, err = repo.FindByID(ctx, user.ID)
		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		tdb.ClearCollection(t, "users")

		nonExistentID := primitive.NewObjectID()
		err := repo.Delete(ctx, nonExistentID)

		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})
}
