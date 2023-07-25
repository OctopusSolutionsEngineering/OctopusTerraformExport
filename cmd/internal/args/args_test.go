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
		"-excludeProjects",
		"test",
		"-excludeTenants",
		"mytenant",
		"-excludeVariableEnvironmentScopes",
		"myenv",
		"-excludeProjectVariable",
		"myvar",
		"-excludeRunbook",
		"myrunbook",
		"-excludeRunbookRegex",
		"myrunbookregex",
		"-excludeLibraryVariableSet",
		"mylibvarset",
		"-excludeLibraryVariableSetRegex",
		"mylibvarsetregex",
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

	if args.ExcludeProjects[0] != "test" {
		t.Fatalf("Project test should have been excluded")
	}

	if args.ExcludeTenants[0] != "mytenant" {
		t.Fatalf("Tenant mytenant should have been excluded")
	}

	if args.ExcludeVariableEnvironmentScopes[0] != "myenv" {
		t.Fatalf("Variable scope myenv should have been excluded")
	}

	if args.ExcludeProjectVariables[0] != "myvar" {
		t.Fatalf("Variable myvar should have been excluded")
	}

	if args.ExcludeRunbooks[0] != "myrunbook" {
		t.Fatalf("Runbook myrunbook should have been excluded")
	}

	if args.ExcludeRunbooksRegex[0] != "myrunbookregex" {
		t.Fatalf("Runbook regex myrunbookregex should have been excluded")
	}

	if args.ExcludeLibraryVariableSets[0] != "mylibvarset" {
		t.Fatalf("Variable set mylibvarset should have been excluded")
	}

	if args.ExcludeLibraryVariableSetsRegex[0] != "mylibvarsetregex" {
		t.Fatalf("Variable set regex mylibvarsetregex should have been excluded")
	}
}
