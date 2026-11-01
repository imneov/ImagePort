package regclient_test

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
	"github.com/imneov/ImagePort/internal/secret"
)

// TestHarborProjectBrowsing exercises authorized product browsing with catalog disabled.
func TestHarborProjectBrowsing(test *testing.T) {
	vault := secret.New()
	if err := vault.Set("test-account", "test-token", true); err != nil {
		test.Fatal(err)
	}
	server := httptest.NewTLSServer(harborHandler(test))
	defer server.Close()
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	profile := domain.Profile{Name: "Harbor fixture", Host: strings.TrimPrefix(server.URL, "https://"), Username: "test-user", SecretRef: "test-account", SessionOnly: true, CAPEM: string(certificate)}
	client := adapter.New(vault)
	first, err := client.Projects(context.Background(), profile, "", 2)
	if err != nil || !reflect.DeepEqual(first.Items, []string{"alpha", "team"}) || first.Next != "2" || first.Total != 3 || !first.HasTotal {
		test.Fatalf("project page: %+v %v", first, err)
	}
	second, err := client.Projects(context.Background(), profile, first.Next, 2)
	if err != nil || !reflect.DeepEqual(second.Items, []string{"zeta"}) || second.Next != "" {
		test.Fatalf("project continuation: %+v %v", second, err)
	}
	repositories, err := client.ProjectRepositories(context.Background(), profile, "team", "", 2)
	if err != nil || !reflect.DeepEqual(repositories.Items, []string{"team/api", "team/worker"}) || repositories.Next != "2" || repositories.Total != 3 {
		test.Fatalf("repository page: %+v %v", repositories, err)
	}
	last, err := client.ProjectRepositories(context.Background(), profile, "team", repositories.Next, 2)
	if err != nil || len(last.Items) != 1 || last.Next != "" {
		test.Fatalf("repository continuation: %+v %v", last, err)
	}
}

// harborHandler verifies Basic authentication, page forwarding and decoded response totals.
func harborHandler(test *testing.T) http.Handler {
	test.Helper()
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, password, valid := request.BasicAuth()
		if !valid || user != "test-user" || password != "test-token" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/api/v2.0/systeminfo" {
			writeHarborJSON(test, writer, map[string]string{"harbor_version": "v2.10.1"})
			return
		}
		if request.URL.Query().Get("page_size") != "2" {
			test.Errorf("lost server limit: %s", request.URL)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		items := []string{"alpha", "team"}
		if request.URL.Path == "/api/v2.0/projects/team/repositories" {
			items = []string{"team/api", "team/worker"}
		} else if request.URL.Path != "/api/v2.0/projects" {
			test.Errorf("unexpected API path: %s", request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if request.URL.Query().Get("page") == "2" {
			items = []string{"zeta"}
		}
		writer.Header().Set("X-Total-Count", "3")
		result := make([]map[string]string, len(items))
		for index, name := range items {
			result[index] = map[string]string{"name": name}
		}
		writeHarborJSON(test, writer, result)
	})
}

// writeHarborJSON reports fixture transport failures without leaking account metadata.
func writeHarborJSON(test *testing.T, writer http.ResponseWriter, value any) {
	test.Helper()
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		test.Error(err)
	}
}

// TestHarborFailures keeps unsupported APIs separate from authentication/permission failures.
func TestHarborFailures(test *testing.T) {
	for status, code := range map[int]string{404: "UNSUPPORTED", 401: "UNAUTHORIZED", 403: "FORBIDDEN"} {
		test.Run(fmt.Sprint(status), func(test *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) { writer.WriteHeader(status) }))
			defer server.Close()
			profile := domain.Profile{Name: "fixture", Host: strings.TrimPrefix(server.URL, "http://"), HTTP: true}
			_, err := adapter.New(secret.New()).Projects(context.Background(), profile, "", 50)
			if !registry.IsCode(err, code) {
				test.Fatalf("incorrect classification: %v", err)
			}
		})
	}
}
