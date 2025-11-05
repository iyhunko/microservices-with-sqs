package sql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		user := &model.User{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "Test User",
			Region:   "US",
			Status:   "active",
			Role:     "user",
		}

		mock.ExpectPrepare("INSERT INTO users").
			ExpectExec().
			WithArgs(sqlmock.AnyArg(), user.Email, user.Password, user.Name, user.Region, user.Status, user.Role, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		result, err := repo.Create(ctx, user)
		require.NoError(t, err)
		assert.NotNil(t, result)

		createdUser := result.(*model.User)
		assert.NotEqual(t, uuid.Nil, createdUser.ID)
		assert.Equal(t, user.Email, createdUser.Email)
		assert.False(t, createdUser.CreatedAt.IsZero())
		assert.False(t, createdUser.UpdatedAt.IsZero())

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("unique constraint violation", func(t *testing.T) {
		user := &model.User{
			Email:    "duplicate@example.com",
			Password: "password123",
			Name:     "Test User",
			Region:   "US",
			Status:   "active",
			Role:     "user",
		}

		pqErr := &pq.Error{
			Code:   pqUniqueViolationErrCode,
			Detail: "Key (email)=(duplicate@example.com) already exists.",
		}

		mock.ExpectPrepare("INSERT INTO users").
			ExpectExec().
			WithArgs(sqlmock.AnyArg(), user.Email, user.Password, user.Name, user.Region, user.Status, user.Role, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(pqErr)

		result, err := repo.Create(ctx, user)
		require.Error(t, err)
		assert.Nil(t, result)

		var uniqueErr *repository.UniqueConstraintError
		assert.ErrorAs(t, err, &uniqueErr)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("successful find", func(t *testing.T) {
		id := uuid.New()

		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "region", "status", "role", "created_at", "updated_at"}).
			AddRow(id, "test@example.com", "password123", "Test User", "US", "active", "user", now, now)

		mock.ExpectPrepare("SELECT \\* FROM users WHERE id = \\$1").
			ExpectQuery().
			WithArgs(id).
			WillReturnRows(rows)

		result, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.NotNil(t, result)

		foundUser := result.(*model.User)
		assert.Equal(t, id, foundUser.ID)
		assert.Equal(t, "test@example.com", foundUser.Email)
		assert.Equal(t, "Test User", foundUser.Name)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectPrepare("SELECT \\* FROM users WHERE id = \\$1").
			ExpectQuery().
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		result, err := repo.FindByID(ctx, id)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "user not found")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("list without filters", func(t *testing.T) {
		query := repository.NewQuery()
		query.Limit = 10

		now := time.Now()
		id1 := uuid.New()
		id2 := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "region", "status", "role", "created_at", "updated_at"}).
			AddRow(id1, "user1@example.com", "pass1", "User 1", "US", "active", "user", now, now).
			AddRow(id2, "user2@example.com", "pass2", "User 2", "EU", "active", "admin", now, now)

		mock.ExpectPrepare("SELECT \\* FROM users WHERE 1=1 ORDER BY created_at DESC, id DESC LIMIT").
			ExpectQuery().
			WithArgs(10).
			WillReturnRows(rows)

		result, err := repo.List(ctx, *query)
		require.NoError(t, err)
		assert.Len(t, result, 2)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with region filter", func(t *testing.T) {
		query := repository.NewQuery().With(repository.Region, "US")
		query.Limit = 10

		now := time.Now()
		id := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "region", "status", "role", "created_at", "updated_at"}).
			AddRow(id, "user1@example.com", "pass1", "User 1", "US", "active", "user", now, now)

		mock.ExpectPrepare("SELECT \\* FROM users WHERE 1=1 AND region = \\$1 ORDER BY created_at DESC, id DESC LIMIT").
			ExpectQuery().
			WithArgs("US", 10).
			WillReturnRows(rows)

		result, err := repo.List(ctx, *query)
		require.NoError(t, err)
		assert.Len(t, result, 1)

		user := result[0].(*model.User)
		assert.Equal(t, "US", user.Region)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_DeleteByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		id := uuid.New()
		user := &model.User{ID: id}

		mock.ExpectPrepare("DELETE FROM users WHERE id").
			ExpectExec().
			WithArgs(id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.DeleteByID(ctx, user)
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		id := uuid.New()
		user := &model.User{ID: id}

		mock.ExpectPrepare("DELETE FROM users WHERE id").
			ExpectExec().
			WithArgs(id).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.DeleteByID(ctx, user)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
