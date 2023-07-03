package args

import (
	"testing"
)

func TestParseFlagsCorrect(t *testing.T) {
	args, _, err := ParseArgs([]string{
		"-url",
		"http://example.org",
		"-space",
		"Spaces-1",
		"-apiKey",
		"API-xxxx",
		"-dest",
		"/tmp",
		"-console",
	})

	if err != nil {
		t.Fatalf("Should not have returned an error")
	}

	if args.Url != "http://example.org" {
		t.Fatalf("Url should have been http://example.org")
	}

	if args.Space != "Spaces-1" {
		t.Fatalf("Space should have been Spaces-1")
	}

	if args.ApiKey != "API-xxxx" {
		t.Fatalf("ApiKey should have been API-xxxx")
	}

	if args.Destination != "/tmp" {
		t.Fatalf("Destination should have been /tmp")
	}

	if !args.Console {
		t.Fatalf("Console should have been true")
	}
}
