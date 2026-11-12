package reports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

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
	// maxContributionBytes caps the multipart body of an aporte: one photo plus
	// room for the text fields.
	maxContributionBytes = MaxPhotoBytes + 64<<10
	// maxJSONBytes caps the JSON body of the similar-reports check: up to 1100
	// characters of text, at most 4 bytes each, plus the coordinates.
	maxJSONBytes = 16 << 10
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
	if req.Params.Category != nil {
		in.Categories = *req.Params.Category
	}
	if req.Params.Status != nil {
		in.Status = string(*req.Params.Status)
	}
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

// GetReportMap implements the getReportMap operation of the contract.
func (h *Handler) GetReportMap(ctx context.Context, req api.GetReportMapRequestObject) (api.GetReportMapResponseObject, error) {
	var in MapInput
	if req.Params.Category != nil {
		in.Categories = *req.Params.Category
	}
	if req.Params.Status != nil {
		in.Status = string(*req.Params.Status)
	}

	pins, err := h.service.Map(ctx, in)
	switch {
	case errors.Is(err, ErrInvalidReport):
		return api.GetReportMap400TextResponse(err.Error()), nil
	case err != nil:
		return nil, err
	}

	items := make([]api.ReportPin, len(pins))
	for i, p := range pins {
		items[i] = api.ReportPin{
			Id:       p.ID,
			Title:    p.Title,
			Category: api.Category{Slug: p.CategorySlug, Label: p.CategoryLabel},
			Status:   api.ReportStatus(p.Status),
			Location: api.Location{Latitude: p.Latitude, Longitude: p.Longitude},
		}
	}
	return api.GetReportMap200JSONResponse(api.ReportMap{Items: items}), nil
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
			AffectedCount:  r.AffectedCount,
		}
	}
	return api.ReportPage{Items: items, NextCursor: optional(page.NextCursor)}
}

// GetReport implements the getReport operation of the contract.
func (h *Handler) GetReport(ctx context.Context, req api.GetReportRequestObject) (api.GetReportResponseObject, error) {
	caller, err := callerID(ctx)
	if err != nil {
		return api.GetReport401TextResponse("unauthorized"), nil
	}
	r, err := h.service.Get(ctx, req.Id, caller)
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

		AffectedCount:     r.AffectedCount,
		ContributionCount: r.ContributionCount,
		Support:           api.SupportState(r.Support),
	}, nil
}

// errNoClaims means the request has no verified session. The auth middleware
// already rejects such requests, so this is a second barrier.
var errNoClaims = errors.New("no session")

// callerID returns the user id of the verified session.
func callerID(ctx context.Context) (uuid.UUID, error) {
	claims, ok := auth.ClaimsFrom(ctx)
	if !ok {
		return uuid.Nil, errNoClaims
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("token subject is not a uuid: %w", err)
	}
	return id, nil
}

// FindSimilarReports implements the findSimilarReports operation of the
// contract.
func (h *Handler) FindSimilarReports(ctx context.Context, req api.FindSimilarReportsRequestObject) (api.FindSimilarReportsResponseObject, error) {
	if _, err := callerID(ctx); errors.Is(err, errNoClaims) {
		return api.FindSimilarReports401TextResponse("unauthorized"), nil
	} else if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return api.FindSimilarReports400TextResponse("body is required"), nil
	}

	// The coordinates are pointers so that a missing one is not read as 0.
	if req.Body.Latitude == nil || req.Body.Longitude == nil {
		return api.FindSimilarReports400TextResponse("latitude and longitude are required"), nil
	}
	in := SimilarReportsInput{Title: req.Body.Title, Latitude: *req.Body.Latitude, Longitude: *req.Body.Longitude}
	if req.Body.Details != nil {
		in.Details = *req.Body.Details
	}
	found, err := h.service.Similar(ctx, in)
	switch {
	case errors.Is(err, ErrInvalidReport):
		return api.FindSimilarReports400TextResponse(err.Error()), nil
	case errors.Is(err, ErrComparisonUnavailable):
		slog.WarnContext(ctx, "similar reports check failed", "error", err)
		return api.FindSimilarReports503TextResponse("comparison unavailable"), nil
	case err != nil:
		return nil, err
	}

	items := make([]api.SimilarReport, len(found))
	for i, r := range found {
		items[i] = api.SimilarReport{
			Id:             r.ID,
			Title:          r.Title,
			Category:       api.Category{Slug: r.CategorySlug, Label: r.CategoryLabel},
			Status:         api.ReportStatus(r.Status),
			CreatedAt:      r.CreatedAt,
			CoverPhoto:     r.CoverPhoto.URL,
			DistanceMeters: r.DistanceMeters,
			AffectedCount:  r.AffectedCount,
		}
	}
	return api.FindSimilarReports200JSONResponse(api.SimilarReports{Items: items}), nil
}

// SupportReport implements the supportReport operation of the contract.
func (h *Handler) SupportReport(ctx context.Context, req api.SupportReportRequestObject) (api.SupportReportResponseObject, error) {
	caller, err := callerID(ctx)
	if errors.Is(err, errNoClaims) {
		return api.SupportReport401TextResponse("unauthorized"), nil
	} else if err != nil {
		return nil, err
	}
	out, err := h.service.Support(ctx, req.Id, caller)
	switch {
	case errors.Is(err, ErrNotFound):
		return api.SupportReport404TextResponse("not found"), nil
	case errors.Is(err, ErrIsAuthor):
		return api.SupportReport409TextResponse(ErrIsAuthor.Error()), nil
	case err != nil:
		return nil, err
	}
	return api.SupportReport200JSONResponse(toAPISupport(out)), nil
}

// UnsupportReport implements the unsupportReport operation of the contract.
func (h *Handler) UnsupportReport(ctx context.Context, req api.UnsupportReportRequestObject) (api.UnsupportReportResponseObject, error) {
	caller, err := callerID(ctx)
	if errors.Is(err, errNoClaims) {
		return api.UnsupportReport401TextResponse("unauthorized"), nil
	} else if err != nil {
		return nil, err
	}
	out, err := h.service.Unsupport(ctx, req.Id, caller)
	switch {
	case errors.Is(err, ErrNotFound):
		return api.UnsupportReport404TextResponse("not found"), nil
	case errors.Is(err, ErrIsAuthor):
		return api.UnsupportReport409TextResponse(ErrIsAuthor.Error()), nil
	case err != nil:
		return nil, err
	}
	return api.UnsupportReport200JSONResponse(toAPISupport(out)), nil
}

func toAPISupport(s ReportSupportDTO) api.ReportSupport {
	return api.ReportSupport{AffectedCount: s.AffectedCount, Support: api.SupportState(s.Support)}
}

// ListReportContributions implements the listReportContributions operation of
// the contract.
func (h *Handler) ListReportContributions(ctx context.Context, req api.ListReportContributionsRequestObject) (api.ListReportContributionsResponseObject, error) {
	if _, err := callerID(ctx); errors.Is(err, errNoClaims) {
		return api.ListReportContributions401TextResponse("unauthorized"), nil
	} else if err != nil {
		return nil, err
	}
	in := ListContributionsInput{ReportID: req.Id}
	if req.Params.Cursor != nil {
		in.Cursor = *req.Params.Cursor
	}
	if req.Params.Limit != nil {
		in.Limit = *req.Params.Limit
	}

	page, err := h.service.Contributions(ctx, in)
	switch {
	case errors.Is(err, ErrNotFound):
		return api.ListReportContributions404TextResponse("not found"), nil
	case errors.Is(err, ErrInvalidReport), errors.Is(err, ErrInvalidCursor):
		return api.ListReportContributions400TextResponse(err.Error()), nil
	case err != nil:
		return nil, err
	}

	items := make([]api.Contribution, len(page.Items))
	for i, c := range page.Items {
		items[i] = toAPIContribution(c)
	}
	return api.ListReportContributions200JSONResponse(api.ContributionPage{
		Items:      items,
		NextCursor: optional(page.NextCursor),
	}), nil
}

// CreateReportContribution implements the createReportContribution operation
// of the contract. Like CreateReport, it reads the multipart body itself.
func (h *Handler) CreateReportContribution(ctx context.Context, req api.CreateReportContributionRequestObject) (api.CreateReportContributionResponseObject, error) {
	authorID, err := callerID(ctx)
	if errors.Is(err, errNoClaims) {
		return api.CreateReportContribution401TextResponse("unauthorized"), nil
	} else if err != nil {
		return nil, err
	}

	in, err := readContributionForm(req.Body)
	switch {
	case bodyExceeded(ctx), errors.Is(err, ErrPhotoTooLarge):
		return api.CreateReportContribution413TextResponse("photo too large"), nil
	case errors.Is(err, ErrInvalidReport):
		return api.CreateReportContribution400TextResponse(err.Error()), nil
	case err != nil:
		return api.CreateReportContribution400TextResponse("invalid multipart body"), nil
	}
	in.ReportID, in.AuthorID = req.Id, authorID

	created, err := h.service.Contribute(ctx, in)
	var limit *TooManyContributionsError
	switch {
	case errors.As(err, &limit):
		wait := int(math.Ceil(limit.RetryAfter.Seconds()))
		return api.CreateReportContribution429TextResponse{
			Body:    "too many contributions",
			Headers: api.CreateReportContribution429ResponseHeaders{RetryAfter: &wait},
		}, nil
	case errors.Is(err, ErrNotFound):
		return api.CreateReportContribution404TextResponse("not found"), nil
	case errors.Is(err, ErrPhotoTooLarge):
		return api.CreateReportContribution413TextResponse("photo too large"), nil
	case errors.Is(err, ErrInvalidReport):
		return api.CreateReportContribution400TextResponse(err.Error()), nil
	case err != nil:
		return nil, err
	}
	return api.CreateReportContribution201JSONResponse(api.CreatedContribution{
		Contribution:  toAPIContribution(created.Contribution),
		AffectedCount: created.AffectedCount,
		Support:       api.SupportState(created.Support),
	}), nil
}

func toAPIContribution(c ContributionDTO) api.Contribution {
	return api.Contribution{
		Id:         c.ID,
		Text:       c.Text,
		AuthorName: optional(c.AuthorName),
		CreatedAt:  c.CreatedAt,
		PhotoUrl:   optional(c.PhotoURL),
	}
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

// readContributionForm reads the multipart parts of an aporte into the service
// input, never holding more than the size limits allow. A second photo is
// rejected: an aporte has at most one.
func readContributionForm(mr *multipart.Reader) (ContributionInput, error) {
	var in ContributionInput
	var hasText bool

	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return in, err
		}

		name := part.FormName()
		switch name {
		case "photo":
			if in.Photo != nil {
				return in, invalid("an aporte has at most one photo")
			}
			data, err := io.ReadAll(io.LimitReader(part, MaxPhotoBytes+1))
			if err != nil {
				return in, err
			}
			if len(data) > MaxPhotoBytes {
				return in, ErrPhotoTooLarge
			}
			in.Photo = data
		case "text", "authorName":
			raw, err := io.ReadAll(io.LimitReader(part, maxFieldBytes+1))
			if err != nil {
				return in, err
			}
			if len(raw) > maxFieldBytes {
				return in, invalid("%s is too long", name)
			}
			if name == "text" {
				in.Text, hasText = string(raw), true
			} else {
				in.AuthorName = string(raw)
			}
		default:
			continue // unknown parts are ignored
		}
	}

	if !hasText {
		return in, invalid("text is required")
	}
	return in, nil
}

type limitedBodyKey struct{}

// bodyLimit is the largest body each operation accepts: an aporte carries one
// photo, the similar-reports check only a short JSON text, and a report up to
// three photos.
func bodyLimit(r *http.Request) int64 {
	if r.Method != http.MethodPost {
		return maxRequestBytes
	}
	switch {
	case strings.HasSuffix(r.URL.Path, "/contributions"):
		return maxContributionBytes
	case r.URL.Path == "/reports/similar":
		return maxJSONBytes
	}
	return maxRequestBytes
}

// limitBody caps the request body so a large upload cannot exhaust memory.
// The multipart reader hides the reader error that says the cap was hit, so
// the capped body is also kept in the context for bodyExceeded.
func limitBody() api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, bodyLimit(r))
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
func Register(r chi.Router, verifier auth.RequestVerifier, repo Repository, photos PhotoStore, embedder Embedder) error {
	spec, err := api.GetSpec()
	if err != nil {
		return err
	}
	strict := api.NewStrictHandlerWithOptions(NewHandler(NewService(repo, photos, embedder)), nil, api.StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: httperror.Handle,
	})
	api.HandlerWithOptions(strict, api.ChiServerOptions{
		BaseRouter:  r,
		Middlewares: []api.MiddlewareFunc{auth.Middleware(spec, verifier), limitBody()},
	})
	return nil
}

var _ api.StrictServerInterface = (*Handler)(nil)
