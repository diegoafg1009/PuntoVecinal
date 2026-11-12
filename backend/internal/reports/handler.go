package reports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/httperror"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/reports/api"
)

const (
	// maxRequestBytes caps the whole multipart body: three photos at the
	// per-photo limit plus room for the text fields.
	maxRequestBytes = 3*MaxPhotoBytes + 1<<20
	// maxFieldBytes caps each text field (details is at most 1000 characters).
	maxFieldBytes = 16 << 10
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// CreateReport implements the createReport operation of the contract. The
// body arrives as a multipart stream: the handler reads it with size limits
// and the service validates every field.
func (h *Handler) CreateReport(ctx context.Context, req api.CreateReportRequestObject) (api.CreateReportResponseObject, error) {
	claims, ok := auth.ClaimsFrom(ctx)
	if !ok {
		return api.CreateReport401TextResponse("unauthorized"), nil
	}
	authorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("token subject is not a uuid: %w", err)
	}

	in, err := readForm(req.Body)
	switch {
	case bodyExceeded(ctx), errors.Is(err, ErrPhotoTooLarge):
		return api.CreateReport413TextResponse("photo too large"), nil
	case errors.Is(err, ErrInvalidReport):
		return api.CreateReport400TextResponse(err.Error()), nil
	case err != nil:
		return api.CreateReport400TextResponse("invalid multipart body"), nil
	}
	in.AuthorID = authorID

	created, err := h.service.Create(ctx, in)
	var limit *TooManyReportsError
	switch {
	case errors.As(err, &limit):
		wait := int(math.Ceil(limit.RetryAfter.Seconds()))
		return api.CreateReport429TextResponse{
			Body:    "too many reports",
			Headers: api.CreateReport429ResponseHeaders{RetryAfter: &wait},
		}, nil
	case errors.Is(err, ErrPhotoTooLarge):
		return api.CreateReport413TextResponse("photo too large"), nil
	case errors.Is(err, ErrInvalidReport):
		return api.CreateReport400TextResponse(err.Error()), nil
	case err != nil:
		return nil, err
	}
	return api.CreateReport201JSONResponse{Id: created.ID, Status: api.CreatedReportStatus(created.Status)}, nil
}

// ListReports implements the listReports operation of the contract.
func (h *Handler) ListReports(ctx context.Context, req api.ListReportsRequestObject) (api.ListReportsResponseObject, error) {
	in := ListReportsInput{Latitude: req.Params.Latitude, Longitude: req.Params.Longitude}
	if req.Params.Cursor != nil {
		in.Cursor = *req.Params.Cursor
	}
	if req.Params.Limit != nil {
		in.Limit = *req.Params.Limit
	}

	page, err := h.service.List(ctx, in)
	switch {
	case errors.Is(err, ErrInvalidReport), errors.Is(err, ErrInvalidCursor):
		return api.ListReports400TextResponse(err.Error()), nil
	case err != nil:
		return nil, err
	}

	return api.ListReports200JSONResponse(toAPIPage(page)), nil
}

// SearchReports implements the searchReports operation of the contract.
func (h *Handler) SearchReports(ctx context.Context, req api.SearchReportsRequestObject) (api.SearchReportsResponseObject, error) {
	p := req.Params
	in := SearchReportsInput{Latitude: p.Latitude, Longitude: p.Longitude}
	if p.Q != nil {
		in.Query = *p.Q
	}
	if p.Sort != nil {
		in.Sort = string(*p.Sort)
	}
	if p.Cursor != nil {
		in.Cursor = *p.Cursor
	}
	if p.Limit != nil {
		in.Limit = *p.Limit
	}

	page, err := h.service.Search(ctx, in)
	switch {
	case errors.Is(err, ErrInvalidReport), errors.Is(err, ErrInvalidCursor):
		return api.SearchReports400TextResponse(err.Error()), nil
	case err != nil:
		return nil, err
	}
	return api.SearchReports200JSONResponse(toAPIPage(page)), nil
}

// toAPIPage maps a page of cards to the contract's ReportPage.
func toAPIPage(page ReportPageDTO) api.ReportPage {
	items := make([]api.ReportSummary, len(page.Items))
	for i, r := range page.Items {
		items[i] = api.ReportSummary{
			Id:             r.ID,
			Title:          r.Title,
			Category:       api.Category{Slug: r.CategorySlug, Label: r.CategoryLabel},
			Status:         api.ReportStatus(r.Status),
			Reference:      optional(r.Reference),
			CreatedAt:      r.CreatedAt,
			CoverPhoto:     api.ReportPhoto{Url: r.CoverPhoto.URL},
			DistanceMeters: r.DistanceMeters,
		}
	}
	return api.ReportPage{Items: items, NextCursor: optional(page.NextCursor)}
}

// GetReport implements the getReport operation of the contract.
func (h *Handler) GetReport(ctx context.Context, req api.GetReportRequestObject) (api.GetReportResponseObject, error) {
	r, err := h.service.Get(ctx, req.Id)
	if errors.Is(err, ErrNotFound) {
		return api.GetReport404TextResponse("not found"), nil
	}
	if err != nil {
		return nil, err
	}

	photos := make([]api.ReportPhoto, len(r.Photos))
	for i, p := range r.Photos {
		photos[i] = api.ReportPhoto{Url: p.URL}
	}
	return api.GetReport200JSONResponse{
		Id:         r.ID,
		Title:      r.Title,
		Details:    optional(r.Details),
		Category:   api.Category{Slug: r.CategorySlug, Label: r.CategoryLabel},
		Status:     api.ReportStatus(r.Status),
		Location:   api.Location{Latitude: r.Latitude, Longitude: r.Longitude},
		Reference:  optional(r.Reference),
		AuthorName: optional(r.AuthorName),
		CreatedAt:  r.CreatedAt,
		Photos:     photos,
	}, nil
}

// optional maps the empty string to null.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// readForm reads the multipart parts into the service input, never holding
// more than the size limits allow.
func readForm(mr *multipart.Reader) (CreateReportInput, error) {
	var in CreateReportInput
	var lat, lng *float64

	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return in, err
		}

		name := part.FormName()
		if name == "photos" {
			if len(in.Photos) == maxPhotos {
				return in, invalid("a report needs between %d and %d photos", minPhotos, maxPhotos)
			}
			data, err := io.ReadAll(io.LimitReader(part, MaxPhotoBytes+1))
			if err != nil {
				return in, err
			}
			if len(data) > MaxPhotoBytes {
				return in, ErrPhotoTooLarge
			}
			in.Photos = append(in.Photos, data)
			continue
		}

		switch name {
		case "category", "title", "details", "reference", "authorName", "latitude", "longitude":
		default:
			continue // unknown parts are ignored
		}
		raw, err := io.ReadAll(io.LimitReader(part, maxFieldBytes+1))
		if err != nil {
			return in, err
		}
		if len(raw) > maxFieldBytes {
			return in, invalid("%s is too long", name)
		}
		value := string(raw)

		switch name {
		case "category":
			in.CategorySlug = value
		case "title":
			in.Title = value
		case "details":
			in.Details = value
		case "reference":
			in.Reference = value
		case "authorName":
			in.AuthorName = value
		case "latitude", "longitude":
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return in, invalid("%s must be a number", name)
			}
			if name == "latitude" {
				lat = &f
			} else {
				lng = &f
			}
		}
	}

	if lat == nil {
		return in, invalid("latitude is required")
	}
	if lng == nil {
		return in, invalid("longitude is required")
	}
	in.Latitude, in.Longitude = *lat, *lng
	return in, nil
}

type limitedBodyKey struct{}

// limitBody caps the request body so a large upload cannot exhaust memory.
// The multipart reader hides the reader error that says the cap was hit, so
// the capped body is also kept in the context for bodyExceeded.
func limitBody(max int64) api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, max)
			ctx := context.WithValue(r.Context(), limitedBodyKey{}, r.Body)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bodyExceeded tells whether reading the request body hit the cap set by
// limitBody. The capped reader keeps returning its error once the cap is hit.
func bodyExceeded(ctx context.Context) bool {
	body, ok := ctx.Value(limitedBodyKey{}).(io.Reader)
	if !ok {
		return false
	}
	var tooBig *http.MaxBytesError
	_, err := body.Read(make([]byte, 1))
	return errors.As(err, &tooBig)
}

// Register mounts the module routes on r, protected as the module's contract
// declares.
func Register(r chi.Router, verifier auth.RequestVerifier, repo Repository, photos PhotoStore) error {
	spec, err := api.GetSpec()
	if err != nil {
		return err
	}
	strict := api.NewStrictHandlerWithOptions(NewHandler(NewService(repo, photos)), nil, api.StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: httperror.Handle,
	})
	api.HandlerWithOptions(strict, api.ChiServerOptions{
		BaseRouter:  r,
		Middlewares: []api.MiddlewareFunc{auth.Middleware(spec, verifier), limitBody(maxRequestBytes)},
	})
	return nil
}

var _ api.StrictServerInterface = (*Handler)(nil)
