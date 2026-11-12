package reports

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StatusPublished is the status every report starts with.
const StatusPublished = "publicado"

var (
	// ErrInvalidReport marks any problem with the data a neighbor sent; the
	// wrapped message says which one.
	ErrInvalidReport = errors.New("invalid report")
	// ErrPhotoTooLarge is returned when a photo exceeds the maximum size.
	ErrPhotoTooLarge = errors.New("photo too large")
	// ErrTooManyReports is returned when a neighbor reached the publication
	// limit. It is always wrapped in a TooManyReportsError.
	ErrTooManyReports = errors.New("too many reports")
	// ErrNotFound is returned when a report does not exist.
	ErrNotFound = errors.New("report not found")
)

// TooManyReportsError tells how long the neighbor has to wait.
type TooManyReportsError struct {
	RetryAfter time.Duration
}

func (e *TooManyReportsError) Error() string {
	return fmt.Sprintf("%v: retry after %s", ErrTooManyReports, e.RetryAfter)
}

func (e *TooManyReportsError) Unwrap() error { return ErrTooManyReports }

// Report is a published report, as it is stored.
type Report struct {
	ID            uuid.UUID
	AuthorID      uuid.UUID // session that published it; never shown, and not read back
	CategorySlug  string
	CategoryLabel string // filled when the report is read back
	Status        string // filled when the report is read back
	CreatedAt     time.Time
	Title         string
	Details       string // empty when not given
	Reference     string // empty when not given
	Latitude      float64
	Longitude     float64
	AuthorName    string // empty means "Vecino anónimo"
	Photos        []Photo
}

// Photo is a stored photo of a report; Position starts at 1.
type Photo struct {
	Position   int
	StorageKey string
}

// RecentReports summarizes the reports an author published inside a window.
type RecentReports struct {
	Count  int
	Oldest time.Time // creation time of the oldest one; zero when Count is 0
}
