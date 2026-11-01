package docker_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// load models daemon JSON status and publishes a target only for a successful archive load.
func (fixture *dockerFixture) load(writer http.ResponseWriter) {
	fixture.loads++
	if fixture.loadError {
		fixture.write(writer, map[string]string{"error": "simulated daemon failure"})
		return
	}
	fixture.tags[fixture.loadTarget] = fixture.id
	fixture.write(writer, map[string]string{"stream": "Loaded image"})
}

// remove rejects destructive flags and removes only the specifically requested alias.
func (fixture *dockerFixture) remove(writer http.ResponseWriter, request *http.Request, path string) {
	if request.URL.Query().Get("force") != "" || request.URL.Query().Get("noprune") != "1" {
		fixture.test.Error("unsafe image removal flags")
	}
	tag := strings.TrimPrefix(path, "/images/")
	delete(fixture.tags, tag)
	fixture.write(writer, []map[string]string{{"Untagged": tag}})
}

// TestLoadAndRemoveVerifyExactAlias exercises SDK status errors, conflict refusal and limited deletion.
func TestLoadAndRemoveVerifyExactAlias(test *testing.T) {
	client, fixture := newDockerFixture(test)
	listing, err := client.Images(context.Background())
	if err != nil {
		test.Fatal(err)
	}
	target := "team/imported:v1"
	fixture.loadTarget = target
	fixture.tags["team/original:v1"] = fixture.id
	if _, err := client.Load(context.Background(), listing.Engine, strings.NewReader("archive"), target, "changed", fixture.id); err == nil {
		test.Fatal("unconfirmed target state accepted")
	}
	if fixture.loads != 0 {
		test.Fatal("conflicting target was written")
	}
	fixture.loadError = true
	if _, err := client.Load(context.Background(), listing.Engine, strings.NewReader("archive"), target, "", fixture.id); err == nil {
		test.Fatal("daemon JSON error was ignored")
	}
	fixture.loadError = false
	if _, err := client.Load(context.Background(), listing.Engine, strings.NewReader("archive"), target, "", fixture.id); err != nil {
		test.Fatal(err)
	}
	if err := client.RemoveTag(context.Background(), listing.Engine, target, fixture.id); err != nil {
		test.Fatal(err)
	}
	fixture.mutex.Lock()
	defer fixture.mutex.Unlock()
	if fixture.tags[target] != "" || fixture.tags["team/original:v1"] != fixture.id {
		test.Fatal("deletion affected a different alias")
	}
}

// trackedArchive checks that importing does not transfer the caller's descriptor ownership.
type trackedArchive struct {
	io.Reader
	closed bool
}

// Close records a caller-owned archive descriptor being closed by the transport.
func (input *trackedArchive) Close() error { input.closed = true; return nil }

// TestLoadRetainsInputOwnership prevents successful loads being reported as failures from double Close.
func TestLoadRetainsInputOwnership(test *testing.T) {
	client, fixture := newDockerFixture(test)
	listing, err := client.Images(context.Background())
	if err != nil {
		test.Fatal(err)
	}
	fixture.loadTarget = "team/ownership:v1"
	input := &trackedArchive{Reader: strings.NewReader("archive")}
	if _, err := client.Load(context.Background(), listing.Engine, input, fixture.loadTarget, "", fixture.id); err != nil {
		test.Fatal(err)
	}
	if input.closed {
		test.Fatal("HTTP transport closed the caller-owned archive")
	}
}
