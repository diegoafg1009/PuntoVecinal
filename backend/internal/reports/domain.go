package reports

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// StatusPublished is the status every report starts with.
const StatusPublished = "publicado"

// Statuses is every report status, in the order a report moves through them.
var Statuses = []string{"publicado", "derivado", "recibido", "en_atencion", "atendido", "cerrado"}

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
	// ErrInvalidCursor is returned when a page cursor is not one the API issued.
	ErrInvalidCursor = errors.New("invalid cursor")
	// ErrIsAuthor is returned when the author of a report tries to change their
	// support: the author always counts as affected.
	ErrIsAuthor = errors.New("author cannot change support")
	// ErrComparisonUnavailable is returned when the similar reports cannot be
	// checked because the text-comparison service failed.
	ErrComparisonUnavailable = errors.New("comparison unavailable")
	// ErrTooManyContributions is returned when a neighbor reached the aporte
	// limit. It is always wrapped in a TooManyContributionsError.
	ErrTooManyContributions = errors.New("too many contributions")
)

// Support states: how a caller relates to a report.
const (
	SupportAuthor    = "author"
	SupportSupporter = "supporter"
	SupportNone      = "none"
)

// TooManyReportsError tells how long the neighbor has to wait.
type TooManyReportsError struct {
	RetryAfter time.Duration
}

func (e *TooManyReportsError) Error() string {
	return fmt.Sprintf("%v: retry after %s", ErrTooManyReports, e.RetryAfter)
}

func (e *TooManyReportsError) Unwrap() error { return ErrTooManyReports }

// TooManyContributionsError tells how long the neighbor has to wait.
type TooManyContributionsError struct {
	RetryAfter time.Duration
}

func (e *TooManyContributionsError) Error() string {
	return fmt.Sprintf("%v: retry after %s", ErrTooManyContributions, e.RetryAfter)
}

func (e *TooManyContributionsError) Unwrap() error { return ErrTooManyContributions }

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

	// Embedding is the meaning of the title and details. It is set when the
	// report is created, and nil when it could not be obtained yet.
	Embedding *pgvector.Vector

	// Filled when the report is read back, for one caller.
	AffectedCount     int  // the author plus every other supporter
	ContributionCount int  // number of aportes
	IsAuthor          bool // the caller published the report
	Supported         bool // the caller marked "También me afecta"
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

// ReportSummary is a report as the feed lists it: what a card shows. It never
// carries the author.
type ReportSummary struct {
	ID            uuid.UUID
	Title         string
	CategorySlug  string
	CategoryLabel string
	Status        string
	Reference     string // empty when not given
	CreatedAt     time.Time
	CoverKey      string // storage key of the first photo
	Distance      *int   // whole meters from the requested location; nil without one
	AffectedCount int    // the author plus every other supporter
}

// ListParams selects one page of the feed. The page continues after the
// (After, AfterID) key when HasAfter is set, and holds at most Limit reports.
type ListParams struct {
	HasAfter  bool
	After     time.Time
	AfterID   uuid.UUID
	Limit     int
	Latitude  *float64 // with Longitude: distances are computed from this point
	Longitude *float64
	// Categories keeps only reports in any of these slugs; empty means every
	// category. Status keeps only that status; empty means every status.
	Categories []string
	Status     string
}

// ReportPin is a report as a point of the map: where it is and what it is, and
// never who published it.
type ReportPin struct {
	ID            uuid.UUID
	Title         string
	CategorySlug  string
	CategoryLabel string
	Status        string
	Latitude      float64
	Longitude     float64
}

// MapParams selects the points of the map: the newest Limit reports that match
// the filters, which mean the same as in ListParams.
type MapParams struct {
	Categories []string
	Status     string
	Limit      int
}

// SearchOrder is how the results of a search are sorted.
type SearchOrder string

const (
	// OrderRecent sorts from newest to oldest.
	OrderRecent SearchOrder = "recent"
	// OrderDistance sorts from the nearest to the farthest, newest first on ties.
	OrderDistance SearchOrder = "distance"
)

// SearchParams selects one page of a search. Words are already split and
// escaped for ILIKE; an empty list matches every report. OrderDistance needs
// Latitude and Longitude. The page continues after the key when HasAfter is
// set: (After, AfterID) for OrderRecent, and (AfterDistance, After, AfterID)
// for OrderDistance. It holds at most Limit reports.
type SearchParams struct {
	Order         SearchOrder
	Words         []string
	HasAfter      bool
	After         time.Time
	AfterID       uuid.UUID
	AfterDistance float64 // exact meters; only for OrderDistance
	Limit         int
	Latitude      *float64
	Longitude     *float64
}

// SearchHit is a search result: the card data plus the exact distance in
// meters, which the cursor of OrderDistance needs. ExactDistance is zero
// without a location.
type SearchHit struct {
	ReportSummary
	ExactDistance float64
}

// SimilarParams selects the open reports near a point whose embedding is within
// MaxDistance (cosine distance) of Embedding. At most 5 come back.
type SimilarParams struct {
	Embedding   pgvector.Vector
	Latitude    float64
	Longitude   float64
	MaxDistance float64
}

// SimilarReport is a report as the similar-reports dialog shows it. It never
// carries the author, nor the details.
type SimilarReport struct {
	ID            uuid.UUID
	Title         string
	CategorySlug  string
	CategoryLabel string
	Status        string
	CreatedAt     time.Time
	CoverKey      string // storage key of the first photo
	Distance      int    // whole meters from the checked location
	AffectedCount int    // the author plus every other supporter
}

// ReportText is what the backfill job needs to embed a report.
type ReportText struct {
	ID      uuid.UUID
	Title   string
	Details string // empty when not given
}

// Contribution is an aporte: a text with an optional photo added to a report.
type Contribution struct {
	ID         uuid.UUID
	ReportID   uuid.UUID
	AuthorID   uuid.UUID // session that added it; never shown, and not read back
	AuthorName string    // empty means "Vecino anónimo"
	Text       string
	PhotoKey   string // empty when it has no photo
	CreatedAt  time.Time
}

// ContributionParams selects one page of the aportes of a report, oldest
// first. The page continues after the (After, AfterID) key when HasAfter is
// set, and holds at most Limit aportes.
type ContributionParams struct {
	ReportID uuid.UUID
	HasAfter bool
	After    time.Time
	AfterID  uuid.UUID
	Limit    int
}
