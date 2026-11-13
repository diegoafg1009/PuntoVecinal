package auth

import "github.com/google/uuid"

// CallerDTO is the caller as the service hands it to the handler. Services
// take and return DTOs, never domain entities.
type CallerDTO struct {
	ID          uuid.UUID
	IsAnonymous bool
}
