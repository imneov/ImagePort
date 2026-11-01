package regclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/secret"
)

// TestServerPagination verifies cursor forwarding without dropping a second page.
func TestServerPagination(test *testing.T) {
	for _, catalog := range []bool{true, false} {
		test.Run(map[bool]string{true: "repositories", false: "tags"}[catalog], func(test *testing.T) {
			server := httptest.NewServer(pagedHandler(test, catalog))
			defer server.Close()
			profile := domain.Profile{Name: "test", Host: strings.TrimPrefix(server.URL, "http://"), HTTP: true}
			client := adapter.New(secret.New())
			read := func(cursor string) (domain.Page, error) {
				if catalog {
					return client.Repositories(context.Background(), profile, cursor, 2)
				}
				return client.Tags(context.Background(), profile, "team/image", cursor, 2)
			}
			first, err := read("")
			if err != nil || !reflect.DeepEqual(first.Items, []string{"one", "two"}) || first.Next != "two" {
				test.Fatalf("incorrect first page: %+v %v", first, err)
			}
			second, err := read(first.Next)
			if err != nil || !reflect.DeepEqual(second.Items, []string{"three"}) || second.Next != "" {
				test.Fatalf("incorrect second page: %+v %v", second, err)
			}
		})
	}
}

// pagedHandler validates the server-side limit and cursor for both registry APIs.
func pagedHandler(test *testing.T, catalog bool) http.Handler {
	test.Helper()
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		path, key := "/v2/team/image/tags/list", "tags"
		if catalog {
			path, key = "/v2/_catalog", "repositories"
		}
		if request.URL.Path != path || request.URL.Query().Get("n") != "2" {
			test.Errorf("incorrect pagination request: %s", request.URL)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		items := []string{"one", "two"}
		if cursor := request.URL.Query().Get("last"); cursor == "two" {
			items = []string{"three"}
		} else if cursor != "" {
			test.Errorf("unexpected cursor: %s", cursor)
		}
		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(map[string]any{"name": "team/image", key: items}); err != nil {
			test.Errorf("write registry response: %v", err)
		}
	})
}
