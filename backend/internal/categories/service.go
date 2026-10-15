// Package categories is the API module for the fixed report categories.
package categories

import (
	"context"
	"fmt"
)

// Repository is what the service needs from the database; implemented by
// repository.PgRepository.
type Repository interface {
	// List returns every category ordered by position.
	List(ctx context.Context) ([]Category, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// List returns every category in display order. It never returns a partial
// list: if the repository fails, so does the call.
func (s *Service) List(ctx context.Context) ([]CategoryDTO, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	dtos := make([]CategoryDTO, len(items))
	for i, item := range items {
		dtos[i] = CategoryDTO{Slug: item.Slug, Label: item.Label}
	}
	return dtos, nil
}
