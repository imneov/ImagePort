package transfer_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
	"github.com/imneov/ImagePort/internal/secret"
	"github.com/imneov/ImagePort/internal/store"
	"github.com/imneov/ImagePort/internal/task"
	"github.com/imneov/ImagePort/internal/testutil"
	"github.com/imneov/ImagePort/internal/transfer"
	"github.com/olareg/olareg"
	"github.com/olareg/olareg/config"
)

type environment struct {
	profile   domain.Profile
	port      *adapter.Client
	runner    *transfer.Runner
	tasks     *task.Manager
	directory string
}

// setup starts a genuine in-process OCI registry and production services without Docker.
func setup(test *testing.T) environment {
	test.Helper()
	server := olareg.New(config.Config{Storage: config.ConfigStorage{StoreType: config.StoreMem}})
	httpServer := httptest.NewServer(withoutCatalog(server))
	test.Cleanup(func() {
		httpServer.Close()
		if err := server.Close(); err != nil {
			test.Error(err)
		}
	})
	directory := test.TempDir()
	database, err := store.Open(filepath.Join(directory, "tasks.db"))
	if err != nil {
		test.Fatal(err)
	}
	profile := domain.Profile{ID: "registry", Name: "test registry", Host: strings.TrimPrefix(httpServer.URL, "http://"), HTTP: true}
	if err := database.SaveProfile(profile); err != nil {
		test.Fatal(err)
	}
	port := adapter.New(secret.New())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := &transfer.Runner{Registry: port, Profiles: database, Limits: archive.DefaultLimits(), Logger: logger}
	manager, err := task.New(database, runner, 2, logger)
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() {
		if err := manager.Close(context.Background()); err != nil {
			test.Error(err)
		}
		if err := database.Close(); err != nil {
			test.Error(err)
		}
	})
	return environment{profile, port, runner, manager, directory}
}

// await returns the first terminal snapshot, failing rather than hanging on regressions.
func await(test *testing.T, manager *task.Manager, id string) domain.Task {
	test.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(15 * time.Second)
	for {
		select {
		case <-deadline:
			test.Fatal("task did not terminate")
		case <-ticker.C:
			item, err := manager.Get(id)
			if err != nil {
				test.Fatal(err)
			}
			if item.State != domain.Queued && !item.State.Active() {
				return item
			}
		}
	}
}

// submit checks a production task's final verified success condition.
func submit(test *testing.T, env environment, spec domain.Spec) domain.Task {
	test.Helper()
	spec.ProfileID, spec.RegistryHost = env.profile.ID, env.profile.Host
	id, err := env.tasks.Submit(spec)
	if err != nil {
		test.Fatal(err)
	}
	result := await(test, env.tasks, id)
	if result.State != domain.Succeeded {
		test.Fatalf("task %s: %s", result.State, result.Current().Error)
	}
	return result
}

// seed imports a validated fixture and returns its inspection and full registry target.
func seed(test *testing.T, env environment, files map[string][]byte, name string) (domain.Archive, string) {
	test.Helper()
	filename := filepath.Join(env.directory, name+".tar")
	testutil.Tar(test, filename, files)
	inspection, err := archive.Inspect(context.Background(), filename, archive.DefaultLimits())
	if err != nil {
		test.Fatal(err)
	}
	target := env.profile.Host + "/team/" + name + ":v1"
	submit(test, env, domain.Spec{Operation: "import", Archive: inspection, Entry: inspection.Entries[0], Target: target})
	return inspection, target
}

// TestOCIWorkflow proves full-index upload, platform export, retagging and readback.
func TestOCIWorkflow(test *testing.T) {
	env := setup(test)
	files, root := testutil.OCI(test, "amd64", "arm64")
	_, target := seed(test, env, files, "multi")
	digest, err := env.port.Digest(context.Background(), env.profile, target)
	if err != nil || digest != root {
		test.Fatalf("index changed: %s %v", digest, err)
	}
	sources, err := env.port.Resolve(context.Background(), env.profile, target)
	if err != nil {
		test.Fatal(err)
	}
	if len(sources) != 2 || sources[0].IndexDigest != root || sources[0].Digest == root {
		test.Fatal("platform resolution incorrect")
	}
	output := filepath.Join(env.directory, "下载 image.tar")
	submit(test, env, domain.Spec{Operation: "download", Source: sources[0], Output: output})
	inspection, err := archive.Inspect(context.Background(), output, archive.DefaultLimits())
	if err != nil {
		test.Fatal(err)
	}
	if inspection.Entries[0].Digest != sources[0].Digest {
		test.Fatal("exported wrong platform")
	}
	copyTarget := env.profile.Host + "/team/multi:copy"
	submit(test, env, domain.Spec{Operation: "copy", Source: sources[0], Target: copyTarget})
	unchanged, err := env.port.Digest(context.Background(), env.profile, target)
	if err != nil || unchanged != root {
		test.Fatal("source tag was modified")
	}
	actual, err := env.port.Digest(context.Background(), env.profile, copyTarget)
	if err != nil || actual != sources[0].Digest {
		test.Fatal("target digest mismatch")
	}
}

// TestDockerImport proves SDK normalization and actual compressed-layer upload.
func TestDockerImport(test *testing.T) {
	env := setup(test)
	_, target := seed(test, env, testutil.Docker(test), "docker")
	sources, err := env.port.Resolve(context.Background(), env.profile, target)
	if err != nil {
		test.Fatal(err)
	}
	if len(sources) != 1 || sources[0].Platform != "linux/amd64" {
		test.Fatalf("wrong source %+v", sources)
	}
}

// TestChangedArchiveAndConflict prevent writes after the user-approved inputs change.
func TestChangedArchiveAndConflict(test *testing.T) {
	env := setup(test)
	files, _ := testutil.OCI(test, "amd64")
	inspection, target := seed(test, env, files, "conflict")
	spec := domain.Spec{Operation: "import", ProfileID: env.profile.ID, RegistryHost: env.profile.Host, Archive: inspection, Entry: inspection.Entries[0], Target: target}
	id, err := env.tasks.Submit(spec)
	if err != nil {
		test.Fatal(err)
	}
	if result := await(test, env.tasks, id); result.State != domain.Failed || !strings.Contains(result.Current().Error, "changed") {
		test.Fatalf("unconfirmed overwrite: %+v", result)
	}
	if err := os.WriteFile(inspection.Path, []byte("changed"), 0600); err != nil {
		test.Fatal(err)
	}
	spec.Target = env.profile.Host + "/team/other:v1"
	id, err = env.tasks.Submit(spec)
	if err != nil {
		test.Fatal(err)
	}
	if result := await(test, env.tasks, id); result.State != domain.Failed {
		test.Fatal("changed archive uploaded")
	}
}

type lostResponse struct{ registry.Port }

// Copy simulates a lost success response after the real registry commits the manifest.
func (port lostResponse) Copy(ctx context.Context, profile domain.Profile, source domain.Source, target string) error {
	if err := port.Port.Copy(ctx, profile, source, target); err != nil {
		return err
	}
	return errors.New("simulated lost response")
}

// TestLostResponse is successful only when independent digest readback proves commit.
func TestLostResponse(test *testing.T) {
	env := setup(test)
	files, _ := testutil.OCI(test, "amd64")
	_, target := seed(test, env, files, "source")
	sources, err := env.port.Resolve(context.Background(), env.profile, target)
	if err != nil {
		test.Fatal(err)
	}
	env.runner.Registry = lostResponse{env.port}
	submit(test, env, domain.Spec{Operation: "copy", Source: sources[0], Target: env.profile.Host + "/team/source:lost"})
}

// TestAuthStatusClassification distinguishes authentication, authorization and absence.
func TestAuthStatusClassification(test *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound} {
		test.Run(http.StatusText(status), func(test *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) { writer.WriteHeader(status) }))
			defer server.Close()
			profile := domain.Profile{Name: "test", Host: strings.TrimPrefix(server.URL, "http://"), HTTP: true}
			port := adapter.New(secret.New())
			_, err := port.Digest(context.Background(), profile, "repo:tag")
			expected := map[int]string{401: "UNAUTHORIZED", 403: "FORBIDDEN", 404: "NOT_FOUND"}[status]
			if !registry.IsCode(err, expected) {
				test.Fatalf("wrong classification: %v", err)
			}
		})
	}
}

// withoutCatalog simulates managed registries that prohibit catalog enumeration.
func withoutCatalog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v2/_catalog" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

// TestDisabledCatalog proves direct platform resolution works without enumeration.
func TestDisabledCatalog(test *testing.T) {
	env := setup(test)
	files, _ := testutil.OCI(test, "amd64")
	_, target := seed(test, env, files, "private")
	if _, err := env.port.Repositories(context.Background(), env.profile, "", 50); !registry.IsCode(err, "NOT_FOUND") {
		test.Fatalf("expected disabled catalog: %v", err)
	}
	sources, err := env.port.Resolve(context.Background(), env.profile, target)
	if err != nil || len(sources) != 1 {
		test.Fatalf("direct access failed: %v", err)
	}
}
