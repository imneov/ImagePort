package ui

import (
	"testing"

	"github.com/imneov/ImagePort/internal/domain"
)

// TestDefaultProjectFollowsSourceNamespace verifies the upload form proposes the source project.
func TestDefaultProjectFollowsSourceNamespace(test *testing.T) {
	cases := []struct {
		kind  string
		path  string
		entry domain.LocationEntry
		want  string
	}{
		{kind: "registry", path: "team/service:1.0", want: "team"},
		{kind: "engine", path: "edge/api:v2", want: "edge"},
		{kind: "registry", path: "single:1.0", want: ""},
	}
	for _, item := range cases {
		review := &copyReview{source: workspaceSource{
			target: domain.Target{Kind: item.kind},
			entry:  domain.LocationEntry{Path: item.path, Name: item.path},
		}}
		if got := review.defaultProject(); got != item.want {
			test.Fatalf("%s %q => %q, want %q", item.kind, item.path, got, item.want)
		}
	}
}

// TestDefaultProjectFromArchiveEntry resolves the project from an archive's repository path.
func TestDefaultProjectFromArchiveEntry(test *testing.T) {
	review := &copyReview{source: workspaceSource{
		target:  domain.Target{Kind: "filesystem"},
		entry:   domain.LocationEntry{Path: "edge/api:v1", Name: "edge/api:v1"},
		archive: domain.Archive{Entries: []domain.ArchiveEntry{{Name: "edge/api:v1"}}},
	}}
	if got := review.defaultProject(); got != "edge" {
		test.Fatalf("archive entry project = %q, want edge", got)
	}
}
