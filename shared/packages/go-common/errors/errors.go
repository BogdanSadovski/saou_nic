package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard error codes used across the application.
const (
	CodeInternal       = "INTERNAL"
	CodeNotFound       = "NOT_FOUND"
	CodeBadRequest     = "BAD_REQUEST"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeConflict       = "CONFLICT"
	CodeValidation     = "VALIDATION_ERROR"
	CodeTimeout        = "TIMEOUT"
	CodeUnavailable    = "UNAVAILABLE"
	CodeRateLimited    = "RATE_LIMITED"
	CodeDuplicate      = "DUPLICATE_ENTRY"
	CodeExpired      = "EXPIRED"
	CodeCancelled    = "CANCELLED"
)

// AppError represents an application-level error with additional context.
type AppError struct {
	// Code is a machine-readable error code.
	Code string `json:"code"`
	// Message is a human-readable error message.
	Message string `json:"message"`
	// Details contains additional error details.
	Details map[string]interface{} `json:"details,omitempty"`
	// HTTPStatus is the HTTP status code to return.
	HTTPStatus int `json:"-"`
	// Err is the underlying error (for wrapping).
	Err error `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetail adds a detail key-value pair to the error.
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithDetails replaces all details with the provided map.
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// WithError sets the underlying wrapped error.
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// New creates a new AppError with the given code and message.
func New(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}
}

// Newf creates a new AppError with a formatted message.
func Newf(code, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: http.StatusInternalServerError,
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(code, message string, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// FromError converts a standard error to an AppError.
// If the error is already an AppError, it returns it as-is.
func FromError(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return &AppError{
		Code:       CodeInternal,
		Message:    err.Error(),
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewInternal creates an internal server error.
func NewInternal(message string) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}
}

// NewInternalf creates an internal server error with a formatted message.
func NewInternalf(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: http.StatusInternalServerError,
	}
}

// WrapInternal wraps an error as an internal server error.
func WrapInternal(err error) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    "internal server error",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewNotFound creates a not found error.
func NewNotFound(resource string, id string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s with id '%s' not found", resource, id),
		HTTPStatus: http.StatusNotFound,
	}
}

// NewNotFoundf creates a not found error with a formatted message.
func NewNotFoundf(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: http.StatusNotFound,
	}
}

// NewBadRequest creates a bad request error.
func NewBadRequest(message string) *AppError {
	return &AppError{
		Code:       CodeBadRequest,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewBadRequestf creates a bad request error with a formatted message.
func NewBadRequestf(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       CodeBadRequest,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewUnauthorized creates an unauthorized error.
func NewUnauthorized(message string) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewForbidden creates a forbidden error.
func NewForbidden(message string) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

// NewConflict creates a conflict error.
func NewConflict(message string) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

// NewValidationError creates a validation error.
func NewValidationError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       CodeValidation,
		Message:    message,
		HTTPStatus: http.StatusUnprocessableEntity,
		Details:    details,
	}
}

// NewDuplicateEntry creates a duplicate entry error.
func NewDuplicateEntry(resource string, field string, value string) *AppError {
	return &AppError{
		Code:    CodeDuplicate,
		Message: fmt.Sprintf("duplicate %s: %s '%s' already exists", resource, field, value),
		Details: map[string]interface{}{
			"resource": resource,
			"field":    field,
			"value":    value,
		},
		HTTPStatus: http.StatusConflict,
	}
}

// NewTimeout creates a timeout error.
func NewTimeout(message string) *AppError {
	return &AppError{
		Code:       CodeTimeout,
		Message:    message,
		HTTPStatus: http.StatusGatewayTimeout,
	}
}

// NewUnavailable creates an unavailable error.
func NewUnavailable(message string) *AppError {
	return &AppError{
		Code:       CodeUnavailable,
		Message:    message,
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

// NewRateLimited creates a rate limited error.
func NewRateLimited(message string) *AppError {
	return &AppError{
		Code:       CodeRateLimited,
		Message:    message,
		HTTPStatus: http.StatusTooManyRequests,
	}
}

// IsNotFound checks if the error is a NotFound error.
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeNotFound
	}
	return false
}

// IsUnauthorized checks if the error is an Unauthorized error.
func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeUnauthorized
	}
	return false
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeValidation
	}
	return false
}

// IsDuplicate checks if the error is a duplicate entry error.
func IsDuplicate(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeDuplicate
	}
	return false
}

// ErrNotFound is a sentinel error for not found cases.
var ErrNotFound = New(CodeNotFound, "resource not found")

// ErrUnauthorized is a sentinel error for unauthorized cases.
var ErrUnauthorized = New(CodeUnauthorized, "unauthorized")

// ErrForbidden is a sentinel error for forbidden cases.
var ErrForbidden = New(CodeForbidden, "access forbidden")

// ErrBadRequest is a sentinel error for bad request cases.
var ErrBadRequest = New(CodeBadRequest, "bad request")

// ErrInternal is a sentinel error for internal server errors.
var ErrInternal = New(CodeInternal, "internal server error")
