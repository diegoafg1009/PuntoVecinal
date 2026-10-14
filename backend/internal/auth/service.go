// Package auth is the API module that tells callers who they are.
package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"

	platformauth "github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
)

// ErrNoCaller is returned when the request carries no verified caller.
var ErrNoCaller = errors.New("no authenticated caller")

// Caller is the user behind a request.
type Caller struct {
	ID          uuid.UUID
	IsAnonymous bool
}

type Service struct{}

func NewService() *Service { return &Service{} }

// Me returns the caller identified by the verified token in ctx.
func (s *Service) Me(ctx context.Context) (Caller, error) {
	claims, ok := platformauth.ClaimsFrom(ctx)
	if !ok {
		return Caller{}, ErrNoCaller
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return Caller{}, err
	}
	return Caller{ID: id, IsAnonymous: claims.IsAnonymous}, nil
}
