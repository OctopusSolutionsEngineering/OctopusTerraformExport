package converters

import "testing"

func TestExcludeNone(t *testing.T) {
	excluder := DefaultExcluder{}

	if excluder.IsResourceExcluded("resource", false, nil, nil) {
		t.Fatalf("Resource must not be excluded")
	}
}

func TestExcludeAll(t *testing.T) {
	excluder := DefaultExcluder{}

	if !excluder.IsResourceExcluded("resource", true, nil, nil) {
		t.Fatalf("Resource must be excluded")
	}
}

func TestExcludeByName(t *testing.T) {
	excluder := DefaultExcluder{}

	if !excluder.IsResourceExcluded("resource", false, []string{"resource"}, nil) {
		t.Fatalf("Resource must be excluded")
	}

	if excluder.IsResourceExcluded("resource", false, []string{"blah"}, nil) {
		t.Fatalf("Resource must not be excluded")
	}

	if excluder.IsResourceExcluded("resource", false, []string{""}, nil) {
		t.Fatalf("Resource must not be excluded")
	}
}

func TestExcludeByRegex(t *testing.T) {
	excluder := DefaultExcluder{}

	if !excluder.IsResourceExcludedWithRegex("resource", false, []string{}, []string{"resource|test"}, nil) {
		t.Fatalf("Resource must be excluded")
	}

	if excluder.IsResourceExcludedWithRegex("blah", false, []string{}, []string{"resource|test"}, nil) {
		t.Fatalf("Resource must not be excluded")
	}

	if excluder.IsResourceExcludedWithRegex("resource", false, []string{}, []string{"invalid("}, nil) {
		t.Fatalf("Resource must not be excluded")
	}
}

func TestExcludeByNameException(t *testing.T) {
	excluder := DefaultExcluder{}

	if excluder.IsResourceExcluded("resource", false, nil, []string{"resource"}) {
		t.Fatalf("Resource must not be excluded")
	}

	if !excluder.IsResourceExcluded("resource", false, nil, []string{"blah"}) {
		t.Fatalf("Resource must be excluded")
	}
}

func TestExcludeByEmptyNameException(t *testing.T) {
	excluder := DefaultExcluder{}

	if excluder.IsResourceExcluded("resource", false, nil, []string{}) {
		t.Fatalf("Resource must not be excluded")
	}

	if !excluder.IsResourceExcluded("resource", false, []string{"resource"}, []string{}) {
		t.Fatalf("Resource must be excluded")
	}
}

func TestExcludeByBlankNameException(t *testing.T) {
	excluder := DefaultExcluder{}

	if !excluder.IsResourceExcluded("resource", false, nil, []string{""}) {
		t.Fatalf("Resource must be excluded")
	}
}

func TestExcludeByNameAndException(t *testing.T) {
	excluder := DefaultExcluder{}

	if !excluder.IsResourceExcluded("resource", false, []string{"resource"}, []string{"resource"}) {
		t.Fatalf("Resource must be excluded")
	}

	if !excluder.IsResourceExcluded("resource", false, []string{"resource"}, []string{"blah"}) {
		t.Fatalf("Resource must be excluded")
	}
}

func TestEmptyName(t *testing.T) {
	excluder := DefaultExcluder{}

	if !excluder.IsResourceExcluded("", false, []string{}, []string{}) {
		t.Fatalf("Resource must be excluded")
	}
}
