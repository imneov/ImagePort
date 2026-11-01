package ui

import (
	"strings"
	"testing"

	"github.com/imneov/ImagePort/internal/domain"
)

// TestCopyWorkspaceKeepsIndependentLocations checks selection, naming, translation and invalidation.
func TestCopyWorkspaceKeepsIndependentLocations(test *testing.T) {
	desktop := testDesktop(test)
	desktop.profiles = []domain.Profile{{ID: "first", Name: "First", Host: "one.example.com"}, {ID: "second", Name: "Second", Host: "two.example.com"}}
	desktop.profile.SetOptions([]string{"First", "Second"})
	view := desktop.copyView
	view.reloadLocations()
	if !view.copy.Disabled() {
		test.Fatal("copy enabled without an image")
	}
	engine := domain.Engine{Context: "fixture", Identity: "engine", Host: "unix:///fixture.sock"}
	view.selectLocal(engine, domain.LocalImage{ID: "sha256:local", Tags: []string{"built:v3"}, Platform: "linux/arm64"})
	view.destination.SetSelectedIndex(1)
	if view.copy.Disabled() || view.repository.Text != "built" || view.tag.Text != "v3" || view.preview.Text != "two.example.com/built:v3" || view.local.ImageID != "sha256:local" {
		test.Fatal("local copy did not bind source ID to independent target name")
	}
	desktop.changeLanguage("zh-Hans")
	if view.copy.Text != "复制镜像" || view.copy.Disabled() || !strings.HasPrefix(view.preview.Text, "two.example.com/") {
		test.Fatal("translation altered copy intent")
	}
	view.sourceLocation.SetSelectedIndex(1)
	if !view.copy.Disabled() || view.local.ImageID != "" {
		test.Fatal("changing source location retained the old image")
	}
	source := domain.Source{Reference: "one.example.com/team/api:v1", Digest: "sha256:manifest", Platform: "linux/amd64"}
	view.selectRemote(desktop.profiles[0], source)
	desktop.profile.SetSelectedIndex(1)
	if profile, valid := desktop.connection(); !valid || profile.ID != "first" {
		test.Fatal("a destination/default profile changed the visible source connection")
	}
	view.destination.SetSelectedIndex(len(desktop.profiles))
	view.directory.SetText(test.TempDir())
	if !view.isArchive() || view.copy.Disabled() || !strings.HasSuffix(view.preview.Text, "image-linux-amd64.tar") {
		test.Fatal("remote archive destination lost explicit platform")
	}
	desktop.changeLanguage("en")
	if !view.isArchive() || view.source.Digest != source.Digest {
		test.Fatal("language switch changed the remote source or destination kind")
	}
}

// TestDuplicateConnectionsAndImportDestination preserve distinct identities across modal flows.
func TestDuplicateConnectionsAndImportDestination(test *testing.T) {
	desktop := testDesktop(test)
	desktop.profiles = []domain.Profile{{ID: "first-account", Name: "Registry", Host: "same.example.com"}, {ID: "second-account", Name: "Registry", Host: "same.example.com"}}
	options := profileOptions(desktop.profiles)
	if options[0] == options[1] {
		test.Fatal("saved connections have ambiguous labels")
	}
	desktop.profile.SetOptions(options)
	desktop.profile.SetSelectedIndex(0)
	desktop.copyView.reloadLocations()
	desktop.copyView.sourceLocation.SetSelectedIndex(1)
	view := &importView{desktop: desktop}
	view.importDestination()
	view.destination.SetSelectedIndex(1)
	profile, valid := view.destinationProfile()
	if !valid || profile.ID != "second-account" {
		test.Fatal("archive upload ignored its selected destination")
	}
	if source, valid := desktop.connection(); !valid || source.ID != "first-account" {
		test.Fatal("archive destination changed remote source")
	}
	desktop.copyView.sourceLocation.SetSelectedIndex(2)
	if source, valid := desktop.connection(); !valid || source.ID != "second-account" {
		test.Fatal("duplicate label redirected to another account")
	}
}
