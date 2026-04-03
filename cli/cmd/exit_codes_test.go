package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCLIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		cliErr   *CLIError
		expected string
	}{
		{
			name:     "Error with message only",
			cliErr:   &CLIError{Message: "test error", ExitCode: ExitError},
			expected: "test error",
		},
		{
			name:     "Error with wrapped error",
			cliErr:   &CLIError{Message: "outer error", ExitCode: ExitError, Err: errors.New("inner error")},
			expected: "outer error: inner error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cliErr.Error())
		})
	}
}

func TestCLIError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	cliErr := &CLIError{Message: "outer", ExitCode: ExitError, Err: innerErr}

	assert.Equal(t, innerErr, cliErr.Unwrap())
	assert.True(t, errors.Is(cliErr, innerErr))
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name         string
		constructor  func(string) *CLIError
		expectedCode int
	}{
		{"NewTimeoutError", NewTimeoutError, ExitTimeout},
		{"NewCancelledError", NewCancelledError, ExitCancelled},
		{"NewNotFoundError", NewNotFoundError, ExitNotFound},
		{"NewUnauthorizedError", NewUnauthorizedError, ExitUnauthorized},
		{"NewResourceBusyError", NewResourceBusyError, ExitResourceBusy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor("test message")
			assert.Equal(t, tt.expectedCode, err.ExitCode)
			assert.Equal(t, "test message", err.Message)
		})
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{
			name:         "Nil error returns success",
			err:          nil,
			expectedCode: ExitSuccess,
		},
		{
			name:         "CLIError returns its exit code",
			err:          &CLIError{Message: "test", ExitCode: ExitTimeout},
			expectedCode: ExitTimeout,
		},
		{
			name:         "Error containing timeout",
			err:          errors.New("operation timeout exceeded"),
			expectedCode: ExitTimeout,
		},
		{
			name:         "Error containing cancelled",
			err:          errors.New("reservation was cancelled"),
			expectedCode: ExitCancelled,
		},
		{
			name:         "Error containing canceled (American spelling)",
			err:          errors.New("operation was canceled"),
			expectedCode: ExitCancelled,
		},
		{
			name:         "Error containing not found",
			err:          errors.New("resource not found"),
			expectedCode: ExitNotFound,
		},
		{
			name:         "Error containing status 404",
			err:          errors.New("api error (status: 404)"),
			expectedCode: ExitNotFound,
		},
		{
			name:         "Error containing unauthorized",
			err:          errors.New("unauthorized access"),
			expectedCode: ExitUnauthorized,
		},
		{
			name:         "Error containing status 401",
			err:          errors.New("api error (status: 401)"),
			expectedCode: ExitUnauthorized,
		},
		{
			name:         "Error containing status 403",
			err:          errors.New("api error (status: 403)"),
			expectedCode: ExitUnauthorized,
		},
		{
			name:         "Error containing authentication",
			err:          errors.New("authentication failed"),
			expectedCode: ExitUnauthorized,
		},
		{
			name:         "Generic error returns ExitError",
			err:          errors.New("something went wrong"),
			expectedCode: ExitError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GetExitCode(tt.err)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

func TestGetExitCode_CaseInsensitive(t *testing.T) {
	// Test that pattern matching is case-insensitive
	tests := []struct {
		err          error
		expectedCode int
	}{
		{errors.New("TIMEOUT exceeded"), ExitTimeout},
		{errors.New("Timeout Exceeded"), ExitTimeout},
		{errors.New("CANCELLED by user"), ExitCancelled},
		{errors.New("NOT FOUND in database"), ExitNotFound},
		{errors.New("UNAUTHORIZED request"), ExitUnauthorized},
	}

	for _, tt := range tests {
		code := GetExitCode(tt.err)
		assert.Equal(t, tt.expectedCode, code, "Failed for error: %s", tt.err.Error())
	}
}
