package regclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/secret"
	"github.com/opencontainers/go-digest"
)

// TestHarborDeleteKeepsOtherTags verifies exact tag API scope and post-delete content readback.
func TestHarborDeleteKeepsOtherTags(test *testing.T) {
	body := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":2},"layers":[]}`)
	expected := digest.FromBytes(body).String()
	var mutex sync.Mutex
	deleted := false
	expectedPath := "/api/v2.0/projects/team/repositories/nested%252Fapi/artifacts/" + expected + "/tags/v1"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()
		switch {
		case request.URL.Path == "/api/v2.0/systeminfo":
			if err := json.NewEncoder(writer).Encode(map[string]string{"harbor_version": "v2.10.1"}); err != nil {
				test.Error(err)
			}
		case request.Method == http.MethodDelete:
			if request.URL.EscapedPath() != expectedPath {
				test.Errorf("wrong deletion scope: %s", request.URL.EscapedPath())
				writer.WriteHeader(400)
				return
			}
			deleted = true
			writer.WriteHeader(http.StatusOK)
		case strings.HasPrefix(request.URL.Path, "/v2/team/nested/api/manifests/"):
			if deleted && strings.HasSuffix(request.URL.Path, "/v1") {
				writer.WriteHeader(http.StatusNotFound)
				return
			}
			writer.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			writer.Header().Set("Docker-Content-Digest", expected)
			if _, err := writer.Write(body); err != nil {
				test.Error(err)
			}
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	profile := domain.Profile{Name: "test", Host: strings.TrimPrefix(server.URL, "http://"), HTTP: true}
	client := adapter.New(secret.New())
	if err := client.DeleteTag(context.Background(), profile, "team/nested/api:v1", "changed"); err == nil {
		test.Fatal("changed remote tag was deleted")
	}
	if err := client.DeleteTag(context.Background(), profile, "team/nested/api:v1", expected); err != nil {
		test.Fatal(err)
	}
	actual, err := client.Digest(context.Background(), profile, "team/nested/api:other")
	if err != nil || actual != expected {
		test.Fatalf("other tag was affected: %s %v", actual, err)
	}
}
