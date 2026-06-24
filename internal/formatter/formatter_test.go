package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		length      int
		dataType    string
		expected    string
		expectError bool
	}{
		{
			name:     "numeric with zeros left",
			value:    "123",
			length:   5,
			dataType: "num",
			expected: "00123",
		},
		{
			name:        "numeric exceeds length",
			value:       "123456",
			length:      5,
			dataType:    "num",
			expectError: true,
		},
		{
			name:     "alpha with spaces right",
			value:    "ABC",
			length:   6,
			dataType: "alfa",
			expected: "ABC   ",
		},
		{
			name:     "alpha exceeds length",
			value:    "ABCDEFGHI",
			length:   5,
			dataType: "alfa",
			expected: "ABCDE",
		},
		{
			name:     "empty numeric",
			value:    "",
			length:   5,
			dataType: "num",
			expected: "00000",
		},
		{
			name:     "empty alpha",
			value:    "",
			length:   5,
			dataType: "alfa",
			expected: "     ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatValue(tt.value, tt.length, tt.dataType)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{
			name:     "zero",
			value:    0,
			expected: "000000000000000",
		},
		{
			name:     "integer",
			value:    1500,
			expected: "000000000150000",
		},
		{
			name:     "with cents",
			value:    1500.75,
			expected: "000000000150075",
		},
		{
			name:     "small value",
			value:    0.01,
			expected: "000000000000001",
		},
		{
			name:     "large value",
			value:    9999999999999.99,
			expected: "999999999999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCurrency(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadLeftZeros(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		length   int
		expected string
	}{
		{
			name:     "pad needed",
			value:    "123",
			length:   5,
			expected: "00123",
		},
		{
			name:     "no pad needed",
			value:    "12345",
			length:   5,
			expected: "12345",
		},
		{
			name:     "truncate",
			value:    "123456",
			length:   5,
			expected: "12345",
		},
		{
			name:     "empty string",
			value:    "",
			length:   3,
			expected: "000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadLeftZeros(tt.value, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		length   int
		expected string
	}{
		{
			name:     "pad needed",
			value:    "ABC",
			length:   6,
			expected: "ABC   ",
		},
		{
			name:     "no pad needed",
			value:    "ABCDE",
			length:   5,
			expected: "ABCDE",
		},
		{
			name:     "truncate",
			value:    "ABCDEFGHI",
			length:   5,
			expected: "ABCDE",
		},
		{
			name:     "empty string",
			value:    "",
			length:   3,
			expected: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadRight(tt.value, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}
