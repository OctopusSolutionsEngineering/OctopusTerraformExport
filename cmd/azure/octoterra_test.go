package main

import (
	"net/url"
	"testing"
)

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
func TestHostIsCloudOrLocalParseUrl(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"http://example.octopus.app:80", true},
		{"https://example.testoctopus.com", true},
		{"https://localhost", true},
		{"https://127.0.0.1", true},
		{"https://example.com", false},
		{"https://192.168.1.1", false},
		{"https://octopus.ngrok.app:443", false},
	}

	for _, test := range tests {
		parsedUrl, err := url.Parse(test.host)

		if err != nil {
			t.Error(err)
		}

		result := hostIsCloudOrLocal(parsedUrl.Hostname())
		if result != test.expected {
			t.Errorf("hostIsCloudOrLocal(%s) = %v; want %v", test.host, result, test.expected)
		}
	}
}
