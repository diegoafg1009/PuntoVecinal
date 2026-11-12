package reports

import (
	"time"

	"github.com/google/uuid"
)

// CreateReportInput is what the service needs to publish a report. Photos
// are the raw files as received, in display order.
type CreateReportInput struct {
	AuthorID     uuid.UUID
	CategorySlug string
	Title        string
	Details      string
	Reference    string
	Latitude     float64
	Longitude    float64
	AuthorName   string
	Photos       [][]byte
}

// CreatedReportDTO is the result of publishing a report.
type CreatedReportDTO struct {
	ID     uuid.UUID
	Status string
}

// ReportDTO is a report as the API returns it. It never carries the id of the
// session that published it.
type ReportDTO struct {
	ID            uuid.UUID
	Title         string
	Details       string // empty when not given
	CategorySlug  string
	CategoryLabel string
	Status        string
	Latitude      float64
	Longitude     float64
	Reference     string // empty when not given
	AuthorName    string // empty means "Vecino anónimo"
	CreatedAt     time.Time
	Photos        []PhotoDTO

	AffectedCount     int    // the author plus every other supporter
	ContributionCount int    // number of aportes
	Support           string // SupportAuthor, SupportSupporter or SupportNone
}

// PhotoDTO is a photo of a report, as a temporary signed link.
type PhotoDTO struct {
	URL string
}

// ListReportsInput is what the service needs to list a page of reports. A
// zero Limit means the default page size.
type ListReportsInput struct {
	Cursor    string // empty for the first page
	Limit     int
	Latitude  *float64
	Longitude *float64
	// Categories keeps only reports in any of these slugs (at most 5, empty
	// means every category). Status keeps only that status (empty means every
	// status).
	Categories []string
	Status     string
}

// MapInput is what the service needs to list the map points; the filters mean
// the same as in ListReportsInput.
type MapInput struct {
	Categories []string
	Status     string
}

// ReportPinDTO is a report as a point of the map. It never carries the author
// name nor the id of the session that published it.
type ReportPinDTO struct {
	ID            uuid.UUID
	Title         string
	CategorySlug  string
	CategoryLabel string
	Status        string
	Latitude      float64
	Longitude     float64
}

// ReportSummaryDTO is a report as a card shows it. It never carries the
// author name nor the id of the session that published it.
type ReportSummaryDTO struct {
	ID             uuid.UUID
	Title          string
	CategorySlug   string
	CategoryLabel  string
	Status         string
	Reference      string // empty when not given
	CreatedAt      time.Time
	CoverPhoto     PhotoDTO
	DistanceMeters *int // nil when no location was given
	AffectedCount  int
}

// ReportPageDTO is one page of the feed.
type ReportPageDTO struct {
	Items      []ReportSummaryDTO
	NextCursor string // empty on the last page
}

// SearchReportsInput is what the service needs to search a page of reports.
// An empty Sort means newest first; a zero Limit means the default page size.
type SearchReportsInput struct {
	Query     string // words separated by spaces
	Sort      string // "recent" or "distance"
	Cursor    string // empty for the first page
	Limit     int
	Latitude  *float64
	Longitude *float64
}

// SimilarReportsInput is what the service needs to look for similar reports: a
// draft's text and location.
type SimilarReportsInput struct {
	Title     string
	Details   string
	Latitude  float64
	Longitude float64
}

// SimilarReportDTO is a similar report as the dialog shows it. It never
// carries the details, the author name nor the id of the session that
// published it.
type SimilarReportDTO struct {
	ID             uuid.UUID
	Title          string
	CategorySlug   string
	CategoryLabel  string
	Status         string
	CreatedAt      time.Time
	CoverPhoto     PhotoDTO
	DistanceMeters int
	AffectedCount  int
}

// ReportSupportDTO is the affected count of a report and the caller's relation
// to it, after a support change.
type ReportSupportDTO struct {
	AffectedCount int
	Support       string // SupportAuthor, SupportSupporter or SupportNone
}

// ContributionInput is what the service needs to add an aporte. ReportID is
// the id as received. Photo is the raw file, nil when there is none.
type ContributionInput struct {
	ReportID   string
	AuthorID   uuid.UUID
	Text       string
	AuthorName string
	Photo      []byte
}

// ContributionDTO is an aporte as the API returns it. It never carries the id
// of the session that added it.
type ContributionDTO struct {
	ID         uuid.UUID
	Text       string
	AuthorName string // empty means "Vecino anónimo"
	CreatedAt  time.Time
	PhotoURL   string // empty when it has no photo
}

// CreatedContributionDTO is the result of adding an aporte: the aporte and the
// support it marked.
type CreatedContributionDTO struct {
	Contribution  ContributionDTO
	AffectedCount int
	Support       string
}

// ListContributionsInput is what the service needs to list a page of aportes.
// A zero Limit means the default page size.
type ListContributionsInput struct {
	ReportID string
	Cursor   string // empty for the first page
	Limit    int
}

// ContributionPageDTO is one page of aportes.
type ContributionPageDTO struct {
	Items      []ContributionDTO
	NextCursor string // empty on the last page
}
