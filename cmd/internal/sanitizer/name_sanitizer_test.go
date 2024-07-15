package sanitizer

import "testing"

func TestSanitizer(t *testing.T) {
	sanitizedName := SanitizeName("A string with weird chars !@#$%^&*()+")
	doubleSanitizedName := SanitizeName(sanitizedName)
	expected := "a_string_with_weird_chars____________"
	if sanitizedName != expected {
		t.Fatal("Result should have been " + expected)
	}

	if sanitizedName != doubleSanitizedName {
		t.Fatal("Should have been able to double sanitize a string with no change")
	}
}

func TestSanitizerWithNil(t *testing.T) {
	sanitizedName := SanitizeNamePointer(nil)
	if sanitizedName != "" {
		t.Fatal("Sanitized name should be empty string")
	}
}

func TestSanitizerWithLeadingNumber(t *testing.T) {
	sanitizedName := SanitizeName("0 my project")
	if sanitizedName != "_0_my_project" {
		t.Fatal("Sanitized name should be _0_my_project")
	}
}
