package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imneov/ImagePort/internal/domain"
)

// TestHarborDataReachesWorkspace verifies product API routing, totals and project context.
func TestHarborDataReachesWorkspace(test *testing.T) {
	desktop := testDesktop(test)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Total-Count", "83")
		var value any = []any{map[string]string{"name": "team/api"}, map[string]string{"name": "team/worker"}}
		switch request.URL.Path {
		case "/api/v2.0/systeminfo":
			value = map[string]string{"harbor_version": "v2.10.1"}
		case "/api/v2.0/projects":
			value = []any{map[string]string{"name": "team"}}
		case "/api/v2.0/projects/team/repositories":
		default:
			test.Errorf("wrong browsing API: %s", request.URL.Path)
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := json.NewEncoder(writer).Encode(value); err != nil {
			test.Error(err)
		}
	}))
	defer server.Close()
	profile := domain.Profile{ID: "fixture", Name: "fixture", Host: strings.TrimPrefix(server.URL, "http://"), HTTP: true}
	projects, mode, err := desktop.browser.readPage(context.Background(), profile, "projects", "", "", "1")
	if err != nil || mode != "projects" || len(projects.Items) != 1 {
		test.Fatalf("project response lost: %+v %s %v", projects, mode, err)
	}
	page, mode, err := desktop.browser.readPage(context.Background(), profile, "repos", "team", "", "1")
	if err != nil || mode != "repos" || page.Next != "2" || page.Total != 83 {
		test.Fatalf("repository response lost: %+v %s %v", page, mode, err)
	}
	view := desktop.browser
	view.current, view.listedMode, view.listedProject = page, mode, "team"
	view.listedProfile = profile.ID
	view.render("worker")
	if len(view.rows) != 1 || view.current.Next != "2" || !strings.Contains(view.summary.Text, "83 total") || view.notice.Text != "Project: team" {
		test.Fatal("workspace altered server metadata while filtering")
	}
}

// TestLocalImageWorkspaceUsesRealInputs retains IDs/tags through filtering and language changes.
func TestLocalImageWorkspaceUsesRealInputs(test *testing.T) {
	desktop := testDesktop(test)
	view := desktop.localImages
	view.loaded = true
	view.engine = domain.Engine{Context: "fixture", Host: "unix:///fixture.sock", Identity: "fixture-endpoint"}
	view.images = []domain.LocalImage{{ID: "sha256:first", Tags: []string{"built:v1", "registry.example.com/team/built:v1"}, Platform: "linux/arm64"}, {ID: "sha256:second"}}
	view.filter.SetText("team/built")
	if len(view.rows) != 1 || view.rows[0].ID != "sha256:first" || len(view.images) != 2 {
		test.Fatal("local filter lost content identity")
	}
	desktop.localImageDetails(view.engine, view.rows[0])
	buttonByText(test, desktop.drawer, "Add tag")
	desktop.changeLanguage("zh-Hans")
	buttonByText(test, desktop.drawer, "新增标签")
	if view.filter.Text != "team/built" || view.rows[0].ID != "sha256:first" {
		test.Fatal("language switch altered local image selection inputs")
	}
}
