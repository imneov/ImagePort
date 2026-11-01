package regclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/secret"
	"github.com/imneov/ImagePort/internal/testutil"
	"github.com/olareg/olareg"
	"github.com/olareg/olareg/config"
)

// TestSameRegistrySeparateAccounts proves one host can read and write using different credentials.
func TestSameRegistrySeparateAccounts(test *testing.T) {
	backend := olareg.New(config.Config{Storage: config.ConfigStorage{StoreType: config.StoreMem}})
	defer func() {
		if err := backend.Close(); err != nil {
			test.Error(err)
		}
	}()
	seedServer := httptest.NewServer(backend)
	defer seedServer.Close()
	vault := secret.New()
	for _, user := range []string{"reader", "writer"} {
		if err := vault.Set(user, user+"-token", true); err != nil {
			test.Fatal(err)
		}
	}
	client := adapter.New(vault)
	files, root := testutil.OCI(test, "amd64", "arm64")
	path := filepath.Join(test.TempDir(), "source.tar")
	testutil.Tar(test, path, files)
	seedProfile := domain.Profile{Name: "seed", Host: strings.TrimPrefix(seedServer.URL, "http://"), HTTP: true}
	if err := client.Import(context.Background(), seedProfile, path, domain.ArchiveEntry{}, "source/image:v1"); err != nil {
		test.Fatal(err)
	}
	authenticated := httptest.NewServer(accountScopedRegistry(backend))
	defer authenticated.Close()
	reader := domain.Profile{ID: "reader", Name: "read", Host: strings.TrimPrefix(authenticated.URL, "http://"), HTTP: true, Username: "reader", SecretRef: "reader", SessionOnly: true}
	writer := reader
	writer.ID, writer.Name, writer.Username, writer.SecretRef = "writer", "write", "writer", "writer"
	sources, err := client.Resolve(context.Background(), reader, "source/image:v1")
	if err != nil || len(sources) != 2 {
		test.Fatalf("source: %+v %v", sources, err)
	}
	source := sources[0]
	source.Digest, source.Platform = source.IndexDigest, "all"
	if err := client.CopyBetween(context.Background(), reader, writer, source, "target/copied:v2"); err != nil {
		test.Fatal(err)
	}
	actual, err := client.Digest(context.Background(), writer, "target/copied:v2")
	if err != nil || actual != root {
		test.Fatalf("target: %s %v", actual, err)
	}
	actual, err = client.Digest(context.Background(), reader, "source/image:v1")
	if err != nil || actual != root {
		test.Fatal("copy changed the multi-platform source")
	}
}

// accountScopedRegistry requires different identities at source and destination on the same origin.
func accountScopedRegistry(backend http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		username, password, valid := request.BasicAuth()
		allowed := valid && password == username+"-token"
		if strings.HasPrefix(request.URL.Path, "/v2/source/") {
			allowed = allowed && username == "reader" && (request.Method == http.MethodGet || request.Method == http.MethodHead)
		}
		if strings.HasPrefix(request.URL.Path, "/v2/target/") {
			allowed = allowed && username == "writer"
		}
		if !allowed {
			writer.Header().Set("WWW-Authenticate", `Basic realm="imageport-test"`)
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		backend.ServeHTTP(writer, request)
	})
}
