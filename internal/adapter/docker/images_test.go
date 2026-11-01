package docker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	adapter "github.com/imneov/ImagePort/internal/adapter/docker"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/opencontainers/go-digest"
)

// dockerFixture represents a daemon with one immutable image and mutable aliases.
type dockerFixture struct {
	test       *testing.T
	mutex      sync.Mutex
	id         string
	tags       map[string]string
	exported   bool
	loadTarget string
	loadError  bool
	loads      int
}

// newDockerFixture configures the production SDK through Docker's documented environment.
func newDockerFixture(test *testing.T) (*adapter.Client, *dockerFixture) {
	test.Helper()
	fixture := &dockerFixture{test: test, id: digest.FromString("image configuration").String(), tags: map[string]string{}}
	server := httptest.NewServer(http.HandlerFunc(fixture.serve))
	test.Cleanup(server.Close)
	test.Setenv("DOCKER_CONTEXT", "")
	test.Setenv("DOCKER_HOST", "tcp://"+strings.TrimPrefix(server.URL, "http://"))
	test.Setenv("DOCKER_API_VERSION", "1.44")
	test.Setenv("DOCKER_TLS_VERIFY", "")
	test.Setenv("DOCKER_CERT_PATH", "")
	return adapter.New(), fixture
}

// serve implements only the image APIs used by the SDK, rejecting other operations.
func (fixture *dockerFixture) serve(writer http.ResponseWriter, request *http.Request) {
	fixture.mutex.Lock()
	defer fixture.mutex.Unlock()
	path := strings.TrimPrefix(request.URL.Path, "/v1.44")
	writer.Header().Set("Content-Type", "application/json")
	switch {
	case path == "/images/load":
		fixture.load(writer)
	case request.Method == http.MethodDelete:
		fixture.remove(writer, request, path)
	case path == "/images/json":
		fixture.write(writer, []any{map[string]any{"Id": fixture.id, "RepoTags": []string{"built:latest"}, "Created": 1700000000, "Size": 1024}})
	case strings.HasSuffix(path, "/json"):
		input := strings.TrimSuffix(strings.TrimPrefix(path, "/images/"), "/json")
		if input != fixture.id && fixture.tags[input] == "" {
			writer.WriteHeader(http.StatusNotFound)
			fixture.write(writer, map[string]string{"message": "No such image"})
			return
		}
		fixture.write(writer, map[string]any{"Id": fixture.id, "RepoTags": []string{"built:latest"}, "Os": "linux", "Architecture": "arm64", "Variant": "v8"})
	case strings.HasSuffix(path, "/tag"):
		if path != "/images/"+fixture.id+"/tag" {
			fixture.test.Error("tag source was not pinned by ID")
		}
		fixture.tags[request.URL.Query().Get("repo")+":"+request.URL.Query().Get("tag")] = fixture.id
		writer.WriteHeader(http.StatusCreated)
	case path == "/images/get":
		if request.URL.Query().Get("names") != fixture.id {
			fixture.test.Error("export followed a mutable tag")
		}
		fixture.exported = true
		if _, err := writer.Write([]byte("exported archive stream")); err != nil {
			fixture.test.Error(err)
		}
	default:
		fixture.test.Errorf("unexpected Docker operation: %s %s", request.Method, path)
		writer.WriteHeader(http.StatusNotFound)
	}
}

// write serializes fixture responses and reports transport errors.
func (fixture *dockerFixture) write(writer http.ResponseWriter, value any) {
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		fixture.test.Error(err)
	}
}

// TestLocalImageWorkflow verifies list, immutable inspect, alias creation and export.
func TestLocalImageWorkflow(test *testing.T) {
	client, fixture := newDockerFixture(test)
	listing, err := client.Images(context.Background())
	if err != nil || len(listing.Images) != 1 || listing.Images[0].ID != fixture.id {
		test.Fatalf("list: %+v %v", listing, err)
	}
	source := domain.LocalSource{Engine: listing.Engine, ImageID: listing.Images[0].ID}
	image, err := client.Inspect(context.Background(), listing.Engine, source.ImageID)
	if err != nil || image.Platform != "linux/arm64/v8" {
		test.Fatalf("inspect: %+v %v", image, err)
	}
	if err := client.Tag(context.Background(), source, "registry.example.com/team/app:v1", ""); err != nil {
		test.Fatal(err)
	}
	identity, err := client.TagID(context.Background(), listing.Engine, "registry.example.com/team/app:v1")
	if err != nil || identity != source.ImageID {
		test.Fatalf("tag result: %s %v", identity, err)
	}
	var output bytes.Buffer
	if err := client.Export(context.Background(), source, &output); err != nil {
		test.Fatal(err)
	}
	if output.String() != "exported archive stream" {
		test.Fatal("export stream was lost")
	}
}

// TestChangedEngineAndTagRejectWrites ensures queued input and local conflicts stay pinned.
func TestChangedEngineAndTagRejectWrites(test *testing.T) {
	client, fixture := newDockerFixture(test)
	listing, err := client.Images(context.Background())
	if err != nil {
		test.Fatal(err)
	}
	source := domain.LocalSource{Engine: listing.Engine, ImageID: fixture.id}
	if err := client.Tag(context.Background(), source, "registry.example.com/team/app:v1", "unexpected-old-id"); err == nil {
		test.Fatal("changed local target was replaced")
	}
	test.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
	var output bytes.Buffer
	if err := client.Export(context.Background(), source, &output); err == nil {
		test.Fatal("changed endpoint was accepted")
	}
	if output.Len() != 0 {
		test.Fatal("export wrote before engine identity was checked")
	}
}
