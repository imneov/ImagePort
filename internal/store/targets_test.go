package store

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/imneov/ImagePort/internal/domain"
)

// TestLegacyAccountsMigrateWithoutRetargetingTasks verifies endpoint grouping and original IDs survive reopen.
func TestLegacyAccountsMigrateWithoutRetargetingTasks(test *testing.T) {
	path := filepath.Join(test.TempDir(), "legacy.db")
	database, err := Open(path)
	if err != nil {
		test.Fatal(err)
	}
	profiles := []domain.Profile{{ID: "reader", Name: "Reader", Host: "registry.example.com", Username: "read", SecretRef: "reader"}, {ID: "writer", Name: "Writer", Host: "registry.example.com", Username: "write", SecretRef: "writer"}}
	for _, profile := range profiles {
		if err := database.SaveProfile(profile); err != nil {
			test.Fatal(err)
		}
	}
	task := domain.Task{ID: "existing", Spec: domain.Spec{ProfileID: "writer", RegistryHost: "registry.example.com", SourceProfileID: "reader"}}
	if err := database.SaveTask(task); err != nil {
		test.Fatal(err)
	}
	if _, err := database.db.Exec("PRAGMA user_version=1"); err != nil {
		test.Fatal(err)
	}
	if err := database.Close(); err != nil {
		test.Fatal(err)
	}
	migrated, err := Open(path)
	if err != nil {
		test.Fatal(err)
	}
	defer func() {
		if err := migrated.Close(); err != nil {
			test.Error(err)
		}
	}()
	targets, err := migrated.Targets()
	if err != nil || len(targets) != 1 || targets[0].Address != "https://registry.example.com" {
		test.Fatalf("targets: %+v %v", targets, err)
	}
	for _, original := range profiles {
		profile, err := migrated.Profile(original.ID)
		if err != nil || profile.TargetID != targets[0].ID || profile.SecretRef != original.SecretRef || profile.Username != original.Username {
			test.Fatalf("account changed: %+v %v", profile, err)
		}
	}
	tasks, err := migrated.Tasks()
	if err != nil || len(tasks) != 1 || !reflect.DeepEqual(tasks[0].Spec, task.Spec) {
		test.Fatal("migration retargeted existing tasks")
	}

}

// TestInitialTargetsAreDatabaseScoped ensures separate data directories initialize independently and deletions persist.
func TestInitialTargetsAreDatabaseScoped(test *testing.T) {
	for range 2 {
		database, err := Open(filepath.Join(test.TempDir(), "locations.db"))
		if err != nil {
			test.Fatal(err)
		}
		initial := []domain.Target{{ID: "local", Name: "Local", Kind: "engine"}}
		changed, err := database.InitializeTargets(initial)
		if err != nil || !changed {
			test.Fatalf("initialization: %v %v", changed, err)
		}
		if err := database.DeleteTarget("local"); err != nil {
			test.Fatal(err)
		}
		changed, err = database.InitializeTargets(initial)
		if err != nil || changed {
			test.Fatal("deleted location was recreated")
		}
		targets, err := database.Targets()
		if err != nil || len(targets) != 0 {
			test.Fatal("deleted target survived")
		}
		if err := database.Close(); err != nil {
			test.Fatal(err)
		}
	}
}
