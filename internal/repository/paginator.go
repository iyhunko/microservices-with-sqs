package repository

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInvalidPaginationToken is returned when a pagination token cannot be decoded.
	ErrInvalidPaginationToken = errors.New("token is invalid")
)

const (
	// DefaultPaginationLimit is the default number of items per page.
	DefaultPaginationLimit = 10
	maxPaginationLimit     = 100
)

// Paginator represents pagination state using cursor-based pagination.
type Paginator struct {
	LastID        uuid.UUID
	LastCreatedAt time.Time
}

// Encode encodes the paginator state into a base64-encoded token.
func (t Paginator) Encode() string {
	key := fmt.Sprintf("%s,%s", t.LastCreatedAt.Format(time.RFC3339Nano), t.LastID)
	return base64.StdEncoding.EncodeToString([]byte(key))
}

// DecodePageToken decodes a base64-encoded pagination token into a Paginator.
func DecodePageToken(encodedToken string) (*Paginator, error) {
	bytes, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 token: %w", err)
	}
	decodedStr := string(bytes)
	tokenParts := strings.Split(decodedStr, ",")
	expectedTokenParts := 2
	if len(tokenParts) != expectedTokenParts {
		return nil, fmt.Errorf("invalid token format: %w", ErrInvalidPaginationToken)
	}

	createdAt, err := time.Parse(time.RFC3339Nano, tokenParts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse token timestamp: %w", err)
	}
	id, err := uuid.Parse(tokenParts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse token ID: %w", err)
	}

	return &Paginator{
		LastID:        id,
		LastCreatedAt: createdAt,
	}, nil
}
