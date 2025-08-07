package maputil

import (
	"testing"
)

func TestNullIfEmptyMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "Empty map returns nil",
			input:    map[string]string{},
			expected: nil,
		},
		{
			name:     "Non-empty map returns original map",
			input:    map[string]string{"key": "value"},
			expected: map[string]string{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NilIfEmptyMap(tt.input)

			if result == nil {
				if tt.expected != nil {
					t.Errorf("expected %v, got nil", tt.expected)
				}
				return
			} else {
				if tt.expected == nil {
					t.Errorf("expected nil, got %v", *result)
					return
				}

				if len(*result) != len(tt.expected) {
					t.Errorf("expected length %d, got %d", len(tt.expected), len(*result))
					return
				}
			}
		})
	}
}

func TestValueOrStringDefault(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]any
		key           string
		defaultValue  string
		expectedValue string
	}{
		{
			name:          "Nil input returns default value",
			input:         nil,
			key:           "key",
			defaultValue:  "default",
			expectedValue: "default",
		},
		{
			name:          "Key exists in map",
			input:         map[string]any{"key": "value"},
			key:           "key",
			defaultValue:  "default",
			expectedValue: "value",
		},
		{
			name:          "Key does not exist in map",
			input:         map[string]any{"other_key": "value"},
			key:           "key",
			defaultValue:  "default",
			expectedValue: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValueOrStringDefault(tt.input, tt.key, tt.defaultValue)
			if result != tt.expectedValue {
				t.Errorf("expected %s, got %s", tt.expectedValue, result)
			}
		})
	}
}

func TestValueOrBoolDefault(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]any
		key           string
		defaultValue  bool
		expectedValue bool
	}{
		{
			name:          "Nil input returns default value",
			input:         nil,
			key:           "key",
			defaultValue:  true,
			expectedValue: true,
		},
		{
			name:          "Key exists in map with true value",
			input:         map[string]any{"key": true},
			key:           "key",
			defaultValue:  false,
			expectedValue: true,
		},
		{
			name:          "Key exists in map with false value",
			input:         map[string]any{"key": false},
			key:           "key",
			defaultValue:  true,
			expectedValue: false,
		},
		{
			name:          "Key does not exist in map",
			input:         map[string]any{"other_key": true},
			key:           "key",
			defaultValue:  false,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValueOrBoolDefault(tt.input, tt.key, tt.defaultValue)
			if result != tt.expectedValue {
				t.Errorf("expected %t, got %t", tt.expectedValue, result)
			}
		})
	}
}
