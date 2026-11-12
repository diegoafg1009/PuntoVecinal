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
