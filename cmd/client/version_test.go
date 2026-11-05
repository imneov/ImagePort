package main

import (
	"testing"

	"fyne.io/fyne/v2"
)

// TestApplicationVersion checks tagged prereleases, numeric package metadata and
// the development fallback without relying on linker flags surviving packaging.
func TestApplicationVersion(test *testing.T) {
	cases := []struct {
		name     string
		metadata fyne.AppMetadata
		expected string
	}{
		{"prerelease", fyne.AppMetadata{Release: true, Version: "0.2.0", Custom: map[string]string{"releaseVersion": "0.2.0-rc.1"}}, "0.2.0-rc.1"},
		{"release", fyne.AppMetadata{Release: true, Version: "0.2.0"}, "0.2.0"},
		{"development", fyne.AppMetadata{Version: "0.0.1"}, version},
		{"empty", fyne.AppMetadata{}, version},
	}
	for _, fixture := range cases {
		test.Run(fixture.name, func(test *testing.T) {
			if actual := applicationVersion(fixture.metadata); actual != fixture.expected {
				test.Fatalf("version = %q, expected %q", actual, fixture.expected)
			}
		})
	}
}
