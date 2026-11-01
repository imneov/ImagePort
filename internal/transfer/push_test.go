package transfer_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
	"github.com/imneov/ImagePort/internal/testutil"
	"github.com/opencontainers/go-digest"
)

// localArchivePort substitutes a daemon export while the registry and task pipeline remain real.
type localArchivePort struct {
	path    string
	source  domain.LocalSource
	failure error
}

// Images is unused by transfer execution and rejects unexpected calls.
func (*localArchivePort) Images(context.Context) (domain.LocalImages, error) {
	return domain.LocalImages{}, errors.New("unexpected listing")
}

// Inspect is unused because task content is already pinned by immutable ID.
func (*localArchivePort) Inspect(context.Context, domain.Engine, string) (domain.LocalImage, error) {
	return domain.LocalImage{}, errors.New("unexpected inspection")
}

// Tag rejects unexpected mutations; pushing does not need to alter local aliases.
func (*localArchivePort) Tag(context.Context, domain.LocalSource, string, string) error {
	return errors.New("push must not mutate local tags")
}

// TagID rejects mutable tag lookups in the transfer path.
func (*localArchivePort) TagID(context.Context, domain.Engine, string) (string, error) {
	return "", errors.New("push must not follow local tags")
}

// Export records the exact requested source and streams the generated archive.
func (port *localArchivePort) Export(ctx context.Context, source domain.LocalSource, writer io.Writer) error {
	port.source = source
	if port.failure != nil {
		return port.failure
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.Open(port.path)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(writer, file)
	return errors.Join(copyErr, file.Close())
}

// TestLocalPushRoundtrip checks Docker/OCI daemon exports through actual remote digest readback.
func TestLocalPushRoundtrip(test *testing.T) {
	for _, format := range []string{"docker", "oci"} {
		test.Run(format, func(test *testing.T) {
			env := setup(test)
			files := testutil.Docker(test)
			if format == "oci" {
				files, _ = testutil.OCI(test, "amd64", "arm64")
			}
			path := filepath.Join(env.directory, "daemon-export.tar")
			testutil.Tar(test, path, files)
			local := &localArchivePort{path: path}
			env.runner.Local = local
			source := domain.LocalSource{Engine: domain.Engine{Context: "fixture", Identity: "test-endpoint", Host: "unix:///test.sock"}, ImageID: digest.FromString("pinned-daemon-image").String()}
			item := submit(test, env, domain.Spec{Operation: "push", Local: source, Target: env.profile.Host + "/team/built:v1"})
			if item.State != domain.Succeeded || item.Current().ExpectedDigest == "" {
				test.Fatalf("unverified push: %+v", item)
			}
			actual, err := env.port.Digest(context.Background(), env.profile, item.Spec.Target)
			if err != nil || actual != item.Current().ExpectedDigest {
				test.Fatalf("remote readback: %s %v", actual, err)
			}
			if local.source != source {
				test.Fatal("push changed immutable daemon source")
			}
		})
	}
}

// TestLocalExportFailureDoesNotPublish proves a failed snapshot never creates a remote tag.
func TestLocalExportFailureDoesNotPublish(test *testing.T) {
	env := setup(test)
	env.runner.Local = &localArchivePort{failure: errors.New("daemon export failed")}
	spec := domain.Spec{Operation: "push", ProfileID: env.profile.ID, RegistryHost: env.profile.Host, Target: env.profile.Host + "/team/not-published:v1"}
	id, err := env.tasks.Submit(spec)
	if err != nil {
		test.Fatal(err)
	}
	item := await(test, env.tasks, id)
	if item.State != domain.Failed || item.Current().ExpectedDigest != "" {
		test.Fatalf("incorrect failure state: %+v", item)
	}
	_, err = env.port.Digest(context.Background(), env.profile, spec.Target)
	if !registry.IsCode(err, "NOT_FOUND") {
		test.Fatalf("export failure published a remote result: %v", err)
	}
}
