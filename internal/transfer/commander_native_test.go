package transfer_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	dockeradapter "github.com/imneov/ImagePort/internal/adapter/docker"
	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
)

// TestRealDockerCommanderRoutes verifies real daemon pull/load/export/copy using an ephemeral registry.
// IMAGEPORT_TEST_DOCKER_IMAGE selects an existing image; only unique test aliases are added and removed.
func TestRealDockerCommanderRoutes(test *testing.T) {
	input := os.Getenv("IMAGEPORT_TEST_DOCKER_IMAGE")
	if input == "" {
		test.Skip("set IMAGEPORT_TEST_DOCKER_IMAGE to an existing small local image")
	}
	env := setup(test)
	port := dockeradapter.New()
	listing, err := port.Images(context.Background())
	if err != nil {
		test.Fatal(err)
	}
	image, err := port.Inspect(context.Background(), listing.Engine, input)
	if err != nil {
		test.Fatal(err)
	}
	env.runner.Local = port
	source := domain.LocalSource{Engine: listing.Engine, ImageID: image.ID}
	target := env.profile.Host + "/native/roundtrip:v1"
	submit(test, env, domain.Spec{Operation: "push", Local: source, Target: target})
	remote, err := env.port.Resolve(context.Background(), env.profile, target)
	if err != nil || len(remote) == 0 {
		test.Fatalf("resolve: %+v %v", remote, err)
	}
	pullAlias := nativeAlias(test, port, listing.Engine, image.ID)
	submit(test, env, domain.Spec{Operation: "pull", Source: remote[0], Target: pullAlias, DestinationEngine: listing.Engine})
	copyAlias := nativeAlias(test, port, listing.Engine, image.ID)
	submit(test, env, domain.Spec{Operation: "copy-local", Local: source, Source: domain.Source{Platform: image.Platform}, Target: copyAlias, DestinationEngine: listing.Engine})
	output := filepath.Join(env.directory, "native-export.tar")
	submit(test, env, domain.Spec{Operation: "export-local", Local: source, Output: output})
	inspection, err := archive.Inspect(context.Background(), output, archive.DefaultLimits())
	if err != nil || len(inspection.Entries) != 1 {
		test.Fatalf("archive: %+v %v", inspection, err)
	}
	loadAlias := nativeAlias(test, port, listing.Engine, image.ID)
	submit(test, env, domain.Spec{Operation: "load", Archive: inspection, Entry: inspection.Entries[0], Source: domain.Source{Platform: image.Platform}, Target: loadAlias, DestinationEngine: listing.Engine})
	actual, err := port.TagID(context.Background(), listing.Engine, input)
	if err != nil || actual != image.ID {
		test.Fatal("source image was changed")
	}
}

// nativeAlias allocates a unique alias and removes exactly that alias after the test.
func nativeAlias(test *testing.T, port *dockeradapter.Client, engine domain.Engine, expected string) string {
	test.Helper()
	id, err := domain.NewID()
	if err != nil {
		test.Fatal(err)
	}
	alias := "imageport-test/" + id + ":verified"
	test.Cleanup(func() {
		actual, err := port.TagID(context.Background(), engine, alias)
		if err != nil {
			test.Error(err)
			return
		}
		if actual == "" {
			return
		}
		if actual != expected {
			test.Error("test alias changed externally; leaving it for inspection")
			return
		}
		if err := port.RemoveTag(context.Background(), engine, alias, expected); err != nil {
			test.Error(err)
		}
	})
	return alias
}
