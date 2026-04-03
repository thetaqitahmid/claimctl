package cmd

import (
	"fmt"
	"os"
	"strings"
)

// Exit codes for the CLI
const (
	ExitSuccess      = 0 // Operation successful
	ExitError        = 1 // General error
	ExitTimeout      = 2 // Timeout waiting for resource
	ExitCancelled    = 3 // Reservation was cancelled
	ExitNotFound     = 4 // Resource/reservation not found
	ExitUnauthorized = 5 // Authentication failed
	ExitResourceBusy = 6 // Resource busy (if --no-queue flag)
)

// CLIError represents an error with an associated exit code
type CLIError struct {
	Message  string
	ExitCode int
	Err      error
}

func (e *CLIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *CLIError) Unwrap() error {
	return e.Err
}

// Error constructors for common scenarios

// NewTimeoutError creates an error for timeout scenarios
func NewTimeoutError(message string) *CLIError {
	return &CLIError{Message: message, ExitCode: ExitTimeout}
}

// NewCancelledError creates an error for cancelled reservations
func NewCancelledError(message string) *CLIError {
	return &CLIError{Message: message, ExitCode: ExitCancelled}
}

// NewNotFoundError creates an error for not found scenarios
func NewNotFoundError(message string) *CLIError {
	return &CLIError{Message: message, ExitCode: ExitNotFound}
}

// NewUnauthorizedError creates an error for authentication failures
func NewUnauthorizedError(message string) *CLIError {
	return &CLIError{Message: message, ExitCode: ExitUnauthorized}
}

// NewResourceBusyError creates an error for busy resource scenarios
func NewResourceBusyError(message string) *CLIError {
	return &CLIError{Message: message, ExitCode: ExitResourceBusy}
}

// GetExitCode determines the appropriate exit code from an error.
// It checks for CLIError types first, then inspects error messages
// for known patterns from API responses.
func GetExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	// Check if it's already a CLIError
	if cliErr, ok := err.(*CLIError); ok {
		return cliErr.ExitCode
	}

	errStr := strings.ToLower(err.Error())

	// Check for timeout patterns
	if strings.Contains(errStr, "timeout") {
		return ExitTimeout
	}

	// Check for cancelled patterns
	if strings.Contains(errStr, "cancelled") || strings.Contains(errStr, "canceled") {
		return ExitCancelled
	}

	// Check for not found patterns
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "status: 404") {
		return ExitNotFound
	}

	// Check for unauthorized patterns
	if strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "authentication") ||
		strings.Contains(errStr, "status: 401") ||
		strings.Contains(errStr, "status: 403") {
		return ExitUnauthorized
	}

	// Default to general error
	return ExitError
}

// ExitWithError prints the error message to stderr and exits with the
// appropriate exit code. This should be used in command RunE functions
// that need specific exit codes.
func ExitWithError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(GetExitCode(err))
}
