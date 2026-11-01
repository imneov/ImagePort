package transfer_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
	"github.com/imneov/ImagePort/internal/store"
	"github.com/imneov/ImagePort/internal/testutil"
	"github.com/olareg/olareg"
	"github.com/olareg/olareg/config"
)

// TestCrossRegistryCopy verifies selected-platform transfer across real independent endpoints.
func TestCrossRegistryCopy(test *testing.T) {
	env := setup(test)
	files, root := testutil.OCI(test, "amd64", "arm64")
	_, original := seed(test, env, files, "original")
	sources, err := env.port.Resolve(context.Background(), env.profile, original)
	if err != nil {
		test.Fatal(err)
	}
	server := olareg.New(config.Config{Storage: config.ConfigStorage{StoreType: config.StoreMem}})
	endpoint := httptest.NewServer(server)
	defer endpoint.Close()
	defer func() {
		if err := server.Close(); err != nil {
			test.Error(err)
		}
	}()
	targetProfile := domain.Profile{ID: "destination", Name: "Destination", Host: strings.TrimPrefix(endpoint.URL, "http://"), HTTP: true}
	database := env.runner.Profiles.(*store.Store)
	if err := database.SaveProfile(targetProfile); err != nil {
		test.Fatal(err)
	}
	spec := domain.Spec{Operation: "copy", SourceProfileID: env.profile.ID, SourceRegistryHost: env.profile.Host, ProfileID: targetProfile.ID, RegistryHost: targetProfile.Host, Source: sources[1], Target: targetProfile.Host + "/other/copied:v2"}
	id, err := env.tasks.Submit(spec)
	if err != nil {
		test.Fatal(err)
	}
	result := await(test, env.tasks, id)
	actual, err := env.port.Digest(context.Background(), targetProfile, spec.Target)
	if result.State != domain.Succeeded || err != nil || actual != sources[1].Digest {
		test.Fatalf("cross-registry result: %+v %s %v", result, actual, err)
	}
	unchanged, err := env.port.Digest(context.Background(), env.profile, original)
	if err != nil || unchanged != root {
		test.Fatal("copy modified the original multi-platform image")
	}
	spec.SourceRegistryHost, spec.Target = "changed.example.com", targetProfile.Host+"/other/rejected:v2"
	id, err = env.tasks.Submit(spec)
	if err != nil {
		test.Fatal(err)
	}
	if item := await(test, env.tasks, id); item.State != domain.Failed {
		test.Fatal("changed source address was accepted")
	}
	if _, err := env.port.Digest(context.Background(), targetProfile, spec.Target); !registry.IsCode(err, "NOT_FOUND") {
		test.Fatalf("rejected copy wrote a tag: %v", err)
	}
}

// TestChangedTaskAccountIsRejected prevents username edits from silently changing an approved operation.
func TestChangedTaskAccountIsRejected(test *testing.T) {
	env := setup(test)
	files, _ := testutil.OCI(test, "amd64")
	inspection, _ := seed(test, env, files, "account-source")
	changed := env.profile
	changed.Username = "another-account"
	if err := env.runner.Profiles.(*store.Store).SaveProfile(changed); err != nil {
		test.Fatal(err)
	}
	target := env.profile.Host + "/team/rejected-account:v1"
	id, err := env.tasks.Submit(domain.Spec{Operation: "import", ProfileID: env.profile.ID, RegistryHost: env.profile.Host,
		AccessPinned: true, ProfileUsername: "", Archive: inspection, Entry: inspection.Entries[0], Target: target})
	if err != nil {
		test.Fatal(err)
	}
	item := await(test, env.tasks, id)
	if item.State != domain.Failed || !strings.Contains(item.Current().Error, "account changed") {
		test.Fatalf("changed identity was used: %+v", item)
	}
	if _, err := env.port.Digest(context.Background(), env.profile, target); !registry.IsCode(err, "NOT_FOUND") {
		test.Fatalf("rejected task wrote a target: %v", err)
	}
}
