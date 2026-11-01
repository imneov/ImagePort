package presenter_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/presenter"
	"github.com/imneov/ImagePort/internal/registry"
	"github.com/imneov/ImagePort/internal/secret"
	"github.com/imneov/ImagePort/internal/store"
)

// projectPort supplies product pagination and rejects accidental fallback to the global catalog.
type projectPort struct {
	registry.Port
	calls []string
}

// Projects returns a real-shaped first product page, retaining the server total and continuation.
func (port *projectPort) Projects(_ context.Context, profile domain.Profile, cursor string, limit int) (domain.Page, error) {
	port.calls = append(port.calls, profile.ID+":"+cursor)
	return domain.Page{Items: []string{"team"}, Total: 83, HasTotal: true, Next: "2"}, nil
}

// ProjectRepositories preserves qualified repository names from the project API.
func (*projectPort) ProjectRepositories(context.Context, domain.Profile, string, string, int) (domain.Page, error) {
	return domain.Page{Items: []string{"team/nested/api"}, Total: 83, HasTotal: true, Next: "2"}, nil
}

// Tags supplies an exact user-selected version name under a qualified image.
func (*projectPort) Tags(context.Context, domain.Profile, string, string, int) (domain.Page, error) {
	return domain.Page{Items: []string{"v1"}, Total: 1, HasTotal: true}, nil
}

// TestProjectPagesAndSeparateAccess validates adapter-to-presenter shape and endpoint account binding.
func TestProjectPagesAndSeparateAccess(test *testing.T) {
	database, err := store.Open(filepath.Join(test.TempDir(), "workspace.db"))
	if err != nil {
		test.Fatal(err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			test.Error(err)
		}
	}()
	port := &projectPort{}
	service := presenter.New(database, secret.New(), port, nil)
	defer service.Close()
	target, err := service.SaveTarget(domain.Target{Name: "Registry", Kind: "registry", Address: "registry.example.com"})
	if err != nil {
		test.Fatal(err)
	}
	account, err := service.SaveProfile(domain.Profile{Name: "Read", TargetID: target.ID, Host: "registry.example.com", Username: "reader", SessionOnly: true}, "token")
	if err != nil {
		test.Fatal(err)
	}
	if _, err := service.AccessProfile(domain.Target{ID: "other", Address: target.Address}, account.ID); err == nil {
		test.Fatal("account rebound to a different target")
	}
	profile, err := service.AccessProfile(target, account.ID)
	if err != nil {
		test.Fatal(err)
	}
	projects, err := service.BrowseLocation(context.Background(), target, profile, "root", "", "2")
	if err != nil || projects.Next != "2" || projects.Total != 83 || projects.Entries[0].Kind != "project" || port.calls[0] != account.ID+":2" {
		test.Fatalf("project page: %+v %v", projects, err)
	}
	images, err := service.BrowseLocation(context.Background(), target, profile, "project", "team", "")
	if err != nil || images.Entries[0].Name != "nested/api" || images.Entries[0].Path != "team/nested/api" {
		test.Fatalf("qualified image: %+v %v", images, err)
	}
	versions, err := service.BrowseLocation(context.Background(), target, profile, "image", "team/nested/api", "")
	if err != nil || versions.Entries[0].Path != "team/nested/api:v1" {
		test.Fatalf("version: %+v %v", versions, err)
	}
	account.Host = "other.example.com"
	if _, err := service.SaveProfile(account, ""); err == nil {
		test.Fatal("saved credential moved to another registry")
	}
}
