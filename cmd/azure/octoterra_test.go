package main

import "testing"

func TestHostIsCloudOrLocal(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"example.octopus.app", true},
		{"example.testoctopus.com", true},
		{"localhost", true},
		{"127.0.0.1", true},
		{"example.com", false},
		{"192.168.1.1", false},
		{"octopus.ngrok.app", false},
	}

	for _, test := range tests {
		result := hostIsCloudOrLocal(test.host)
		if result != test.expected {
			t.Errorf("hostIsCloudOrLocal(%s) = %v; want %v", test.host, result, test.expected)
		}
	}
}
