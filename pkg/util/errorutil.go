package util

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

// DomainError standardizes application errors.
type DomainError struct {
	Code       string
	Message    string
	HTTPStatus int
	Details    map[string]any
	Err        error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewDomainError constructs a DomainError.
func NewDomainError(code, message string, status int, details map[string]any) *DomainError {
	return &DomainError{Code: code, Message: message, HTTPStatus: status, Details: details}
}

func NewValidationError(message string, details map[string]any) error {
	return NewDomainError("VALIDATION_FAILED", message, http.StatusBadRequest, details)
}

func NewNotFound(resource string, details map[string]any) error {
	if details == nil {
		details = map[string]any{}
	}
	return &DomainError{
		Code:       "NOT_FOUND",
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
		Details:    details,
	}
}

func NewUnauthorized(message string) error {
	return NewDomainError("UNAUTHORIZED", message, http.StatusUnauthorized, nil)
}

func NewForbidden(message string) error {
	return NewDomainError("FORBIDDEN", message, http.StatusForbidden, nil)
}

func NewConflict(message string, details map[string]any) error {
	return NewDomainError("CONFLICT", message, http.StatusConflict, details)
}

func NewInternalError(err error) error {
	return &DomainError{
		Code:       "INTERNAL_ERROR",
		Message:    "internal server error",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// MapError converts generic errors to DomainError.
func ToDomainError(err error) *DomainError {
	if err == nil {
		return nil
	}
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr
	}
	if errors.Is(err, sql.ErrNoRows) {
		if de, ok := NewNotFound("resource", nil).(*DomainError); ok {
			return de
		}
	}
	if de, ok := NewInternalError(err).(*DomainError); ok {
		return de
	}
	return &DomainError{
		Code:       "INTERNAL_ERROR",
		Message:    "internal server error",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

func MapError(err error) error {
	return ToDomainError(err)
}
