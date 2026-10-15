// Package auth is the API module that tells callers who they are.
package auth

import (
	"context"

	"github.com/google/uuid"

	platformauth "github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
)

type Service struct{}

func NewService() *Service { return &Service{} }

// Me returns the caller identified by the verified token in ctx.
func (s *Service) Me(ctx context.Context) (CallerDTO, error) {
	claims, ok := platformauth.ClaimsFrom(ctx)
	if !ok {
		return CallerDTO{}, ErrNoCaller
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return CallerDTO{}, err
	}
	caller := Caller{ID: id, IsAnonymous: claims.IsAnonymous}
	return CallerDTO(caller), nil
}
