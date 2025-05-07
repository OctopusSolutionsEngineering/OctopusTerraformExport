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
