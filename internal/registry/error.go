package registry

import (
	"context"
	"crypto/x509"
	"errors"
	"io/fs"
	"strings"

	"github.com/regclient/regclient/types/errs"
)

// Error is safe to persist or show; the upstream error and auth headers are omitted.
type Error struct {
	Code, Message, Action string
	Retryable             bool
}

// Error returns a user-readable reason and next action without raw transport data.
func (failure *Error) Error() string { return failure.Message + "; " + failure.Action }

// IsCode tests a stable application error code through wrapped errors.
func IsCode(err error, code string) bool {
	var failure *Error
	return errors.As(err, &failure) && failure.Code == code
}

// Safe maps SDK/transport errors to non-secret application errors.
func Safe(err error) error {
	if err == nil {
		return nil
	}
	var known *Error
	if errors.As(err, &known) {
		return known
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	message := strings.ToLower(err.Error()) // Used for classification only, never displayed.
	var unknownCA x509.UnknownAuthorityError
	switch {
	case errors.Is(err, errs.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		return &Error{"NOT_FOUND", "The image, tag, or API does not exist", "Check the repository path and tag", false}
	case strings.Contains(message, "403"), strings.Contains(message, "forbidden"):
		return &Error{"FORBIDDEN", "This account is not authorized for this operation", "Ask the registry administrator for the required repository permissions", false}
	case errors.Is(err, errs.ErrHTTPUnauthorized), errors.Is(err, errs.ErrEmptyChallenge),
		errors.Is(err, errs.ErrNoNewChallenge), errors.Is(err, errs.ErrNoLogin), strings.Contains(message, "401"):
		return &Error{"UNAUTHORIZED", "Authentication failed or sign-in is required", "Check your username, password, or access token", false}
	case errors.Is(err, errs.ErrHTTPRateLimit), strings.Contains(message, "429"):
		return &Error{"RATE_LIMIT", "The registry is rate limiting requests", "Retry manually later", true}
	case errors.As(err, &unknownCA), strings.Contains(message, "certificate"), strings.Contains(message, "x509"):
		return &Error{"TLS", "Certificate verification failed", "Check the system clock or import the correct CA for this registry", false}
	case errors.Is(err, errs.ErrDigestMismatch):
		return &Error{"DIGEST", "The image content digest does not match", "Check the source content before trying again", false}
	default:
		return &Error{"REGISTRY", "The registry operation did not complete", "Check the address, network, environment proxy, and registry compatibility before retrying", true}
	}
}
