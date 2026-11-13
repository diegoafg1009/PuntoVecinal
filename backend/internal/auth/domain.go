package auth

import (
	"errors"

	"github.com/google/uuid"
)

// ErrNoCaller is returned when the request carries no verified caller.
var ErrNoCaller = errors.New("no authenticated caller")

// Caller is the user behind a request.
type Caller struct {
	ID          uuid.UUID
	IsAnonymous bool
}
