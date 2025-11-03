package sql

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	user := &model.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Password:  "hashedpassword",
		Name:      "Test User",
		Region:    "US",
		Status:    "active",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectPrepare("INSERT INTO users").
		ExpectExec().
		WithArgs(user.ID, user.Email, user.Password, user.Name, user.Region, user.Status, user.Role, user.CreatedAt, user.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, user)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Find(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	user := &model.User{ID: userID}

	rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "region", "status", "role", "created_at", "updated_at"}).
		AddRow(userID, "test@example.com", "hashedpassword", "Test User", "US", "active", "user", time.Now(), time.Now())

	mock.ExpectPrepare("SELECT (.+) FROM users WHERE id").
		ExpectQuery().
		WithArgs(userID).
		WillReturnRows(rows)

	found, err := repo.Find(ctx, user)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "test@example.com", user.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Find_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	user := &model.User{ID: userID}

	mock.ExpectPrepare("SELECT (.+) FROM users WHERE id").
		ExpectQuery().
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "name", "region", "status", "role", "created_at", "updated_at"}))

	found, err := repo.Find(ctx, user)
	assert.NoError(t, err)
	assert.False(t, found)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Patch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	user := &model.User{
		ID:        uuid.New(),
		Email:     "updated@example.com",
		Password:  "newhashedpassword",
		Name:      "Updated User",
		Region:    "EU",
		Status:    "active",
		Role:      "admin",
		UpdatedAt: time.Now(),
	}

	mock.ExpectPrepare("UPDATE users").
		ExpectExec().
		WithArgs(user.ID, user.Email, user.Password, user.Name, user.Region, user.Status, user.Role, user.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	updated, err := repo.Patch(ctx, user)
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	user := &model.User{ID: userID}

	mock.ExpectPrepare("DELETE FROM users WHERE id").
		ExpectExec().
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(ctx, user)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	user1ID := uuid.New()
	user2ID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "region", "status", "role", "created_at", "updated_at"}).
		AddRow(user1ID, "user1@example.com", "pass1", "User 1", "US", "active", "user", now, now).
		AddRow(user2ID, "user2@example.com", "pass2", "User 2", "EU", "active", "user", now, now)

	query := repository.NewQuery()
	query.Limit = 10

	mock.ExpectPrepare("SELECT (.+) FROM users WHERE 1=1 ORDER BY created_at, id LIMIT").
		ExpectQuery().
		WithArgs(10).
		WillReturnRows(rows)

	var users []*model.User
	err = repo.List(ctx, &users, *query)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "user1@example.com", users[0].Email)
	assert.Equal(t, "user2@example.com", users[1].Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}
