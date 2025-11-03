package repository

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPaginator(t *testing.T) {
	t.Run("should fail empty token", func(t *testing.T) {
		// given
		pageToken := ""

		// when
		paginator, err := DecodePageToken(pageToken)

		// then
		assert.True(t, errors.Is(err, ErrInvalidPaginationToken))
		assert.Nil(t, paginator)
	})

	t.Run("should fail invalid token", func(t *testing.T) {
		// given
		pageToken := "querty123"

		// when
		paginator, err := DecodePageToken(pageToken)

		// then
		assert.Error(t, err)
		var corruptInputErr base64.CorruptInputError
		assert.True(t, errors.As(err, &corruptInputErr))
		assert.Nil(t, paginator)
	})

	t.Run("should succeed", func(t *testing.T) {
		// given
		originalPaginator := Paginator{
			LastID:        uuid.New(),
			LastCreatedAt: time.Now(),
		}

		// when
		encodedToken := originalPaginator.Encode()
		decodedPaginator, err := DecodePageToken(encodedToken)

		// then
		assert.NoError(t, err)
		assert.Equal(t, originalPaginator.LastID, decodedPaginator.LastID)
		assert.Equal(t, originalPaginator.LastCreatedAt.Unix(), decodedPaginator.LastCreatedAt.Unix())
	})
}
