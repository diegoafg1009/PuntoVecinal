package categories

// CategoryDTO is a category as the service hands it to the handler. Services
// take and return DTOs, never domain entities.
type CategoryDTO struct {
	Slug  string
	Label string
}
