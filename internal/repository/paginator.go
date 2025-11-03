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
	ErrInvalidPaginationToken = errors.New("token is invalid")
)

const (
	DefaultPaginationLimit = 50
	maxPaginationLimit     = 1000
)

type Paginator struct {
	LastID        uuid.UUID
	LastCreatedAt time.Time
}

func (t Paginator) Encode() string {
	key := fmt.Sprintf("%s,%s", t.LastCreatedAt.Format(time.RFC3339Nano), t.LastID)
	return base64.StdEncoding.EncodeToString([]byte(key))
}

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
