package hcl

import "testing"

func TestIsInterpolation(t *testing.T) {
	if IsInterpolation("$${hi}") {
		t.Fatal("String should not be considered interpolated")
	}

	if IsInterpolation("{hi}") {
		t.Fatal("String should not be considered interpolated")
	}

	if IsInterpolation("hi") {
		t.Fatal("String should not be considered interpolated")
	}

	if IsInterpolation("hi ${hi}") {
		t.Fatal("String should not be considered interpolated")
	}

	if IsInterpolation("${hi") {
		t.Fatal("String should not be considered interpolated")
	}

	if !IsInterpolation("${hi}") {
		t.Fatal("String should be considered interpolated")
	}
}

func TestRemoveInterpolation(t *testing.T) {
	if RemoveInterpolation("${hi}") != "hi" {
		t.Fatal("Interpolation removal failed")
	}

	if RemoveInterpolation("${resource.blah}") != "resource.blah" {
		t.Fatal("Interpolation removal failed")
	}

	if RemoveInterpolation("resource.blah") != "resource.blah" {
		t.Fatal("Interpolation removal failed")
	}
}
