package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLabelFilter(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		labels   []string
		expected bool
		hasErr   bool
	}{
		{
			name:     "empty expression",
			expr:     "",
			labels:   []string{"foo"},
			expected: true, // Nil expr matches implicitly if calling logic handles nil
			hasErr:   false,
		},
		{
			name:     "single label match",
			expr:     "dev",
			labels:   []string{"dev", "linux"},
			expected: true,
		},
		{
			name:     "single label mismatch",
			expr:     "prod",
			labels:   []string{"dev", "linux"},
			expected: false,
		},
		{
			name:     "and match",
			expr:     "dev AND linux",
			labels:   []string{"dev", "linux"},
			expected: true,
		},
		{
			name:     "and mismatch",
			expr:     "dev AND windows",
			labels:   []string{"dev", "linux"},
			expected: false,
		},
		{
			name:     "or match",
			expr:     "dev OR windows",
			labels:   []string{"linux", "windows"},
			expected: true,
		},
		{
			name:     "or mismatch",
			expr:     "dev OR windows",
			labels:   []string{"prod", "linux"},
			expected: false,
		},
		{
			name:     "not match",
			expr:     "NOT dev",
			labels:   []string{"prod"},
			expected: true,
		},
		{
			name:     "not mismatch",
			expr:     "NOT dev",
			labels:   []string{"dev", "prod"},
			expected: false,
		},
		{
			name:     "complex expression 1",
			expr:     "(dev OR qa) AND linux",
			labels:   []string{"qa", "linux"},
			expected: true,
		},
		{
			name:     "complex expression 2",
			expr:     "(dev OR qa) AND linux",
			labels:   []string{"dev", "windows"},
			expected: false,
		},
		{
			name:     "quoted string",
			expr:     "'meeting room' AND projector",
			labels:   []string{"Meeting Room", "projector"},
			expected: true,
		},
		{
			name:     "unbalanced parens",
			expr:     "(dev AND linux",
			hasErr:   true,
		},
		{
			name:     "invalid token",
			expr:     "dev XOR linux",
			hasErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := ParseLabelFilter(tc.expr)
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.expr == "" {
					assert.Nil(t, expr)
				} else {
					assert.NotNil(t, expr)
					assert.Equal(t, tc.expected, expr.Matches(tc.labels))
				}
			}
		})
	}
}
