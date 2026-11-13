// Package repository is the database access of the categories module. The
// sqlc/ folder is generated; this file adapts it to the module's domain.
package repository

import (
	"context"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/categories"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/categories/repository/sqlc"
)

// PgRepository reads categories from PostgreSQL through the sqlc queries. It
// satisfies categories.Repository, which the service defines.
type PgRepository struct {
	queries *sqlc.Queries
}

func NewPgRepository(db sqlc.DBTX) *PgRepository { return &PgRepository{queries: sqlc.New(db)} }

// List returns every category ordered by position.
func (r *PgRepository) List(ctx context.Context) ([]categories.Category, error) {
	rows, err := r.queries.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]categories.Category, len(rows))
	for i, row := range rows {
		items[i] = categories.Category{Slug: row.Slug, Label: row.Label}
	}
	return items, nil
}
