package docker_test

import (
	"context"
	"testing"

	"github.com/docker/cli/cli/config"
	contextdocker "github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
)

// TestNamedEnvironmentContext ensures a valid context name never collides with env mode.
func TestNamedEnvironmentContext(test *testing.T) {
	client, _ := newDockerFixture(test)
	listing, err := client.Images(context.Background())
	if err != nil {
		test.Fatal(err)
	}
	directory, previous := test.TempDir(), config.Dir()
	config.SetDir(directory)
	test.Cleanup(func() { config.SetDir(previous) })
	contexts := store.New(config.ContextStoreDir(), store.NewConfig(nil, store.EndpointTypeGetter("docker", func() any { return &contextdocker.EndpointMeta{} })))
	if err := contexts.CreateOrUpdate(store.Metadata{Name: "environment", Endpoints: map[string]any{"docker": contextdocker.EndpointMeta{Host: listing.Engine.Host}}}); err != nil {
		test.Fatal(err)
	}
	test.Setenv("DOCKER_CONTEXT", "environment")
	test.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
	result, err := client.Images(context.Background())
	if err != nil || len(result.Images) != 1 || result.Engine.Context != "environment" || result.Engine.FromEnv {
		test.Fatalf("wrong named endpoint: %+v %v", result.Engine, err)
	}
}

// TestDockerTransportTrust rejects insecure remote configuration before network access.
func TestDockerTransportTrust(test *testing.T) {
	client, _ := newDockerFixture(test)
	test.Setenv("DOCKER_HOST", "tcp://docker.example.invalid:2375")
	if _, err := client.Images(context.Background()); err == nil {
		test.Fatal("plaintext remote endpoint accepted")
	}
	test.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
	test.Setenv("DOCKER_CERT_PATH", test.TempDir())
	if _, err := client.Images(context.Background()); err == nil {
		test.Fatal("TLS configured without verification was accepted")
	}
}
