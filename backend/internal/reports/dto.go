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
}

// ReportPageDTO is one page of the feed.
type ReportPageDTO struct {
	Items      []ReportSummaryDTO
	NextCursor string // empty on the last page
}
