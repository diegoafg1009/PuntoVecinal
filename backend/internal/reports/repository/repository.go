// Package repository is the database access of the reports module. The
// sqlc/ folder is generated; this file adapts it to the module's domain.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/reports"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/reports/repository/sqlc"
)

// PgRepository stores reports in PostgreSQL through the sqlc queries. It
// satisfies reports.Repository, which the service defines.
type PgRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool, queries: sqlc.New(pool)}
}

// CategoryExists tells whether slug is one of the fixed categories.
func (r *PgRepository) CategoryExists(ctx context.Context, slug string) (bool, error) {
	return r.queries.CategoryExists(ctx, slug)
}

// UnknownCategories returns the slugs that are not one of the fixed categories.
func (r *PgRepository) UnknownCategories(ctx context.Context, slugs []string) ([]string, error) {
	return r.queries.UnknownCategories(ctx, slugs)
}

// CountRecent summarizes the reports authorID published since the given time.
func (r *PgRepository) CountRecent(ctx context.Context, authorID uuid.UUID, since time.Time) (reports.RecentReports, error) {
	row, err := r.queries.CountRecentReports(ctx, sqlc.CountRecentReportsParams{
		AuthorID: pgUUID(authorID),
		Since:    pgtype.Timestamptz{Time: since, Valid: true},
	})
	if err != nil {
		return reports.RecentReports{}, err
	}
	recent := reports.RecentReports{Count: int(row.Total)}
	if row.Oldest.Valid {
		recent.Oldest = row.Oldest.Time
	}
	return recent, nil
}

// Create stores the report and its photos in one transaction.
func (r *PgRepository) Create(ctx context.Context, report reports.Report) error {
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		if err := q.InsertReport(ctx, sqlc.InsertReportParams{
			ID:           pgUUID(report.ID),
			AuthorID:     pgUUID(report.AuthorID),
			CategorySlug: report.CategorySlug,
			Title:        report.Title,
			Details:      pgText(report.Details),
			Reference:    pgText(report.Reference),
			Longitude:    report.Longitude,
			Latitude:     report.Latitude,
			AuthorName:   pgText(report.AuthorName),
		}); err != nil {
			return fmt.Errorf("insert report: %w", err)
		}
		for _, p := range report.Photos {
			if err := q.InsertReportPhoto(ctx, sqlc.InsertReportPhotoParams{
				ReportID:   pgUUID(report.ID),
				Position:   int16(p.Position),
				StorageKey: p.StorageKey,
			}); err != nil {
				return fmt.Errorf("insert photo %d: %w", p.Position, err)
			}
		}
		return nil
	})
	return err
}

// Get returns the report with its photos in order, or reports.ErrNotFound.
func (r *PgRepository) Get(ctx context.Context, id uuid.UUID) (reports.Report, error) {
	row, err := r.queries.GetReport(ctx, pgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return reports.Report{}, reports.ErrNotFound
	}
	if err != nil {
		return reports.Report{}, err
	}
	photos, err := r.queries.ListReportPhotos(ctx, pgUUID(id))
	if err != nil {
		return reports.Report{}, fmt.Errorf("list photos: %w", err)
	}

	report := reports.Report{
		ID:            row.ID.Bytes,
		CategorySlug:  row.CategorySlug,
		CategoryLabel: row.CategoryLabel,
		Status:        string(row.Status),
		CreatedAt:     row.CreatedAt.Time,
		Title:         row.Title,
		Details:       row.Details.String,
		Reference:     row.Reference.String,
		Latitude:      row.Latitude,
		Longitude:     row.Longitude,
		AuthorName:    row.AuthorName.String,
		Photos:        make([]reports.Photo, len(photos)),
	}
	for i, p := range photos {
		report.Photos[i] = reports.Photo{Position: int(p.Position), StorageKey: p.StorageKey}
	}
	return report, nil
}

// List returns up to params.Limit reports, newest first, continuing after the
// key in params.
func (r *PgRepository) List(ctx context.Context, params reports.ListParams) ([]reports.ReportSummary, error) {
	arg := sqlc.ListReportsParams{
		RowLimit:   int32(params.Limit),
		Categories: nonNil(params.Categories),
		Status:     nullStatus(params.Status),
	}
	if params.HasAfter {
		arg.AfterCreatedAt = pgtype.Timestamptz{Time: params.After, Valid: true}
		arg.AfterID = pgUUID(params.AfterID)
	}
	if params.Latitude != nil && params.Longitude != nil {
		arg.Lat = pgtype.Float8{Float64: *params.Latitude, Valid: true}
		arg.Lng = pgtype.Float8{Float64: *params.Longitude, Valid: true}
	}
	rows, err := r.queries.ListReports(ctx, arg)
	if err != nil {
		return nil, err
	}
	items := make([]reports.ReportSummary, len(rows))
	for i, row := range rows {
		items[i] = reports.ReportSummary{
			ID:            row.ID.Bytes,
			Title:         row.Title,
			CategorySlug:  row.CategorySlug,
			CategoryLabel: row.CategoryLabel,
			Status:        string(row.Status),
			Reference:     row.Reference.String,
			CreatedAt:     row.CreatedAt.Time,
			CoverKey:      row.CoverKey,
		}
		// The query answers -1 when no location was given.
		if row.DistanceMeters >= 0 {
			meters := int(row.DistanceMeters)
			items[i].Distance = &meters
		}
	}
	return items, nil
}

// ListPins returns up to params.Limit map points, newest first.
func (r *PgRepository) ListPins(ctx context.Context, params reports.MapParams) ([]reports.ReportPin, error) {
	rows, err := r.queries.ListReportPins(ctx, sqlc.ListReportPinsParams{
		Categories: nonNil(params.Categories),
		Status:     nullStatus(params.Status),
		RowLimit:   int32(params.Limit),
	})
	if err != nil {
		return nil, err
	}
	pins := make([]reports.ReportPin, len(rows))
	for i, row := range rows {
		pins[i] = reports.ReportPin{
			ID:            row.ID.Bytes,
			Title:         row.Title,
			CategorySlug:  row.CategorySlug,
			CategoryLabel: row.CategoryLabel,
			Status:        string(row.Status),
			Latitude:      row.Latitude,
			Longitude:     row.Longitude,
		}
	}
	return pins, nil
}

// Search returns up to params.Limit reports that contain every word, in the
// order of params, continuing after the key in params.
func (r *PgRepository) Search(ctx context.Context, params reports.SearchParams) ([]reports.SearchHit, error) {
	words := params.Words
	if words == nil {
		words = []string{}
	}
	var after pgtype.Timestamptz
	var afterID pgtype.UUID
	if params.HasAfter {
		after = pgtype.Timestamptz{Time: params.After, Valid: true}
		afterID = pgUUID(params.AfterID)
	}

	if params.Order == reports.OrderDistance {
		if params.Latitude == nil || params.Longitude == nil {
			return nil, errors.New("search by distance needs a location")
		}
		arg := sqlc.SearchReportsNearestParams{
			Lat:            *params.Latitude,
			Lng:            *params.Longitude,
			Words:          words,
			AfterCreatedAt: after,
			AfterID:        afterID,
			RowLimit:       int32(params.Limit),
		}
		if params.HasAfter {
			arg.AfterDistance = pgtype.Float8{Float64: params.AfterDistance, Valid: true}
		}
		rows, err := r.queries.SearchReportsNearest(ctx, arg)
		if err != nil {
			return nil, err
		}
		hits := make([]reports.SearchHit, len(rows))
		for i, row := range rows {
			meters := int(row.DistanceMeters)
			hits[i] = reports.SearchHit{
				ReportSummary: reports.ReportSummary{
					ID:            row.ID.Bytes,
					Title:         row.Title,
					CategorySlug:  row.CategorySlug,
					CategoryLabel: row.CategoryLabel,
					Status:        string(row.Status),
					Reference:     row.Reference.String,
					CreatedAt:     row.CreatedAt.Time,
					CoverKey:      row.CoverKey,
					Distance:      &meters,
				},
				ExactDistance: row.Distance,
			}
		}
		return hits, nil
	}

	arg := sqlc.SearchReportsRecentParams{
		Words:          words,
		AfterCreatedAt: after,
		AfterID:        afterID,
		RowLimit:       int32(params.Limit),
	}
	if params.Latitude != nil && params.Longitude != nil {
		arg.Lat = pgtype.Float8{Float64: *params.Latitude, Valid: true}
		arg.Lng = pgtype.Float8{Float64: *params.Longitude, Valid: true}
	}
	rows, err := r.queries.SearchReportsRecent(ctx, arg)
	if err != nil {
		return nil, err
	}
	hits := make([]reports.SearchHit, len(rows))
	for i, row := range rows {
		hits[i].ReportSummary = reports.ReportSummary{
			ID:            row.ID.Bytes,
			Title:         row.Title,
			CategorySlug:  row.CategorySlug,
			CategoryLabel: row.CategoryLabel,
			Status:        string(row.Status),
			Reference:     row.Reference.String,
			CreatedAt:     row.CreatedAt.Time,
			CoverKey:      row.CoverKey,
		}
		// The query answers -1 when no location was given.
		if row.DistanceMeters >= 0 {
			meters := int(row.DistanceMeters)
			hits[i].Distance = &meters
		}
	}
	return hits, nil
}

// nonNil turns a nil slice into an empty one: the queries read an empty array
// as "no filter", and a nil slice would be sent as NULL.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// nullStatus maps the empty string to NULL, which means "any status".
func nullStatus(status string) sqlc.NullReportStatus {
	return sqlc.NullReportStatus{ReportStatus: sqlc.ReportStatus(status), Valid: status != ""}
}

func pgUUID(id uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: id, Valid: true} }

// pgText maps the empty string to NULL.
func pgText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
