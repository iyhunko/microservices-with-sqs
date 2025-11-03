package sql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These are integration tests that require a database connection
// They will be skipped if DB is not available

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Skip if running in CI without database
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would need a real database connection for integration testing
	// For now, we'll skip these tests unless explicitly configured
	t.Skip("Integration tests require a configured database")

	return nil
}

func TestResourceRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	user := &model.User{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Region:   "US",
		Status:   "active",
		Role:     "user",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestResourceRepository_Find(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &model.User{
		Email:    "find@example.com",
		Password: "password123",
		Name:     "Find User",
		Region:   "EU",
		Status:   "active",
		Role:     "user",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Find the user
	foundUser := &model.User{ID: user.ID}
	found, err := repo.Find(ctx, foundUser)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, user.Email, foundUser.Email)
	assert.Equal(t, user.Name, foundUser.Name)
}

func TestResourceRepository_List(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	// Create some users
	for i := 0; i < 3; i++ {
		user := &model.User{
			Email:    uuid.New().String() + "@example.com",
			Password: "password123",
			Name:     "List User",
			Region:   "US",
			Status:   "active",
			Role:     "user",
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)
	}

	// List users
	var users []*model.User
	query := repository.NewQuery()
	query.Limit = 10

	err := repo.List(ctx, &users, *query)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 3)
}

func TestResourceRepository_Patch(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	// Create a user
	user := &model.User{
		Email:    "patch@example.com",
		Password: "password123",
		Name:     "Patch User",
		Region:   "US",
		Status:   "active",
		Role:     "user",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Update the user
	user.Name = "Updated Name"
	user.UpdatedAt = time.Now()

	updated, err := repo.Patch(ctx, user)
	require.NoError(t, err)
	assert.True(t, updated)

	// Verify the update
	foundUser := &model.User{ID: user.ID}
	found, err := repo.Find(ctx, foundUser)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Updated Name", foundUser.Name)
}

func TestResourceRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	// Create a user
	user := &model.User{
		Email:    "delete@example.com",
		Password: "password123",
		Name:     "Delete User",
		Region:   "US",
		Status:   "active",
		Role:     "user",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Delete the user
	err = repo.Delete(ctx, user)
	require.NoError(t, err)

	// Verify deletion
	foundUser := &model.User{ID: user.ID}
	found, err := repo.Find(ctx, foundUser)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestResourceRepository_Transaction(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	t.Run("successful transaction", func(t *testing.T) {
		err := repo.Transaction(ctx, func(ctx context.Context, txRepo repository.Repository) error {
			user := &model.User{
				Email:    "tx@example.com",
				Password: "password123",
				Name:     "TX User",
				Region:   "US",
				Status:   "active",
				Role:     "user",
			}
			return txRepo.Create(ctx, user)
		})
		require.NoError(t, err)
	})

	t.Run("rollback on error", func(t *testing.T) {
		err := repo.Transaction(ctx, func(ctx context.Context, txRepo repository.Repository) error {
			user := &model.User{
				Email:    "rollback@example.com",
				Password: "password123",
				Name:     "Rollback User",
				Region:   "US",
				Status:   "active",
				Role:     "user",
			}
			if err := txRepo.Create(ctx, user); err != nil {
				return err
			}
			// Force an error to trigger rollback
			return assert.AnError
		})
		require.Error(t, err)
	})
}
