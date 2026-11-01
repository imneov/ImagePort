package transfer_test

import (
	"context"
	"os"
	"testing"

	dockeradapter "github.com/imneov/ImagePort/internal/adapter/docker"
	"github.com/imneov/ImagePort/internal/domain"
)

// TestRealDockerPush opt-in checks an existing local image against an ephemeral registry only.
func TestRealDockerPush(test *testing.T) {
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
	item := submit(test, env, domain.Spec{Operation: "push", Local: domain.LocalSource{Engine: listing.Engine, ImageID: image.ID}, Target: env.profile.Host + "/native/selected:v1"})
	if item.Current().ExpectedDigest == "" {
		test.Fatal("no verified manifest digest was captured")
	}
	test.Logf("verified real daemon export and loopback registry push: %s", item.Current().ExpectedDigest)
}
