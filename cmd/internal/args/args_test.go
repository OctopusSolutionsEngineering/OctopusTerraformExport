package args

import (
	"testing"
)

func TestParseFlagsCorrect(t *testing.T) {
	args, _, err := ParseArgs([]string{
		"-url",
		"https://example.org",
		"-space",
		"Spaces-1",
		"-apiKey",
		"API-xxxx",
		"-dest",
		"/tmp",
		"-console",
		"-excludeProjects",
		"test",
		"-excludeProjects",
		" ",
		"-excludeTenants",
		"mytenant",
		"-excludeTenants",
		" ",
		"-excludeVariableEnvironmentScopes",
		"myenv",
		"-excludeVariableEnvironmentScopes",
		" ",
		"-excludeProjectVariable",
		"myvar",
		"-excludeProjectVariable",
		" ",
		"-excludeRunbook",
		"myrunbook",
		"-excludeRunbook",
		" ",
		"-excludeRunbookRegex",
		"myrunbookregex",
		"-excludeRunbookRegex",
		" ",
		"-excludeLibraryVariableSet",
		"mylibvarset",
		"-excludeLibraryVariableSet",
		"   ",
		"-excludeLibraryVariableSetRegex",
		"mylibvarsetregex",
		"-excludeLibraryVariableSetRegex",
		"  ",
		"-excludeTenantsExcept",
		"mytenant",
		"-excludeTenantsExcept",
		"  ",
		"-excludeTenantsWithTag",
		"tag/a",
		"-excludeTenantTags",
		"tag/a",
		"-excludeTenantTagSets",
		"tag",
		"-dummySecretVariableValues",
		"-excludeProjectsRegex",
		".*",
		"-excludeProjectsExcept",
		"Test",
		"-excludeAllProjects",
		"-excludeTargets",
		"Test",
		"-excludeTargetsRegex",
		".*",
		"-excludeTargetsExcept",
		"Test",
		"-excludeTenantsRegex",
		".*",
		"-excludeLibraryVariableSetsExcept",
		"Test",
	})

	if err != nil {
		t.Fatalf("Should not have returned an error")
	}

	if args.Url != "https://example.org" {
		t.Fatalf("Url should have been https://example.org")
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

	if len(args.ExcludeProjects) != 1 {
		t.Fatalf("Only one project should be excluded")
	}

	if args.ExcludeProjects[0] != "test" {
		t.Fatalf("Project test should have been excluded")
	}

	if len(args.ExcludeTenants) != 1 {
		t.Fatalf("Only one tenent should be excluded")
	}

	if args.ExcludeTenants[0] != "mytenant" {
		t.Fatalf("Tenant mytenant should have been excluded")
	}

	if len(args.ExcludeProjectVariables) != 1 {
		t.Fatalf("Only one envionment variable scope should be excluded")
	}

	if args.ExcludeVariableEnvironmentScopes[0] != "myenv" {
		t.Fatalf("Variable scope myenv should have been excluded")
	}

	if len(args.ExcludeProjectVariables) != 1 {
		t.Fatalf("Only one project variable should be excluded")
	}

	if args.ExcludeProjectVariables[0] != "myvar" {
		t.Fatalf("Variable myvar should have been excluded")
	}

	if len(args.ExcludeRunbooks) != 1 {
		t.Fatalf("Only one runbook should be excluded")
	}

	if args.ExcludeRunbooks[0] != "myrunbook" {
		t.Fatalf("Runbook myrunbook should have been excluded")
	}

	if len(args.ExcludeRunbooksRegex) != 1 {
		t.Fatalf("Only one runbook should be excluded via regex")
	}

	if args.ExcludeRunbooksRegex[0] != "myrunbookregex" {
		t.Fatalf("Runbook regex myrunbookregex should have been excluded")
	}

	if len(args.ExcludeLibraryVariableSets) != 1 {
		t.Fatalf("Only one library variable set should be excluded")
	}

	if args.ExcludeLibraryVariableSets[0] != "mylibvarset" {
		t.Fatalf("Variable set mylibvarset should have been excluded")
	}

	if len(args.ExcludeLibraryVariableSetsRegex) != 1 {
		t.Fatalf("Only one library variable set should be excluded via regex")
	}

	if args.ExcludeLibraryVariableSetsRegex[0] != "mylibvarsetregex" {
		t.Fatalf("Variable set regex mylibvarsetregex should have been excluded")
	}

	if len(args.ExcludeTenantsExcept) != 1 {
		t.Fatalf("Only one tenent should be excluded")
	}

	if args.ExcludeTenantsExcept[0] != "mytenant" {
		t.Fatalf("Tenants except mytenant should have been excluded")
	}

	if len(args.ExcludeTenantsWithTags) != 1 {
		t.Fatalf("Only one tenent tag should be excluded")
	}

	if args.ExcludeTenantsWithTags[0] != "tag/a" {
		t.Fatalf("Tenants except those with tag tag/a should have been excluded")
	}

	if len(args.ExcludeTenantTags) != 1 {
		t.Fatalf("Only one tenent tag should be excluded")
	}

	if args.ExcludeTenantTags[0] != "tag/a" {
		t.Fatalf("Tag tag/a should have been excluded")
	}

	if len(args.ExcludeTenantTagSets) != 1 {
		t.Fatalf("Only one tag set should be excluded")
	}

	if args.ExcludeTenantTagSets[0] != "tag" {
		t.Fatalf("TagSets tag should have been excluded")
	}

	if !args.DummySecretVariableValues {
		t.Fatalf("dummy secret variables should have been set")
	}

	if args.ExcludeProjectsRegex[0] != ".*" {
		t.Fatalf("exclude projects regex should have been set")
	}

	if args.ExcludeProjectsExcept[0] != "Test" {
		t.Fatalf("exclude projects should have been set")
	}

	if !args.ExcludeAllProjects {
		t.Fatalf("exclude all projects should have been set")
	}

	if args.ExcludeTargets[0] != "Test" {
		t.Fatalf("exclude targets should have been set")
	}

	if args.ExcludeTargetsExcept[0] != "Test" {
		t.Fatalf("exclude targets except should have been set")
	}

	if args.ExcludeTargetsRegex[0] != ".*" {
		t.Fatalf("exclude targets regex should have been set")
	}

	if args.ExcludeTenantsRegex[0] != ".*" {
		t.Fatalf("exclude tenants regex should have been set")
	}

	if args.ExcludeLibraryVariableSetsExcept[0] != "Test" {
		t.Fatalf("exclude library variable sets except should have been set")
	}
}
