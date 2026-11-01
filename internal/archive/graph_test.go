package archive_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/testutil"
)

// TestRepeatedGraphDescriptor validates size even when a digest was already memoized.
func TestRepeatedGraphDescriptor(test *testing.T) {
	files, childDigest := testutil.OCI(test, "amd64")
	childPath := "blobs/sha256/" + childDigest[len("sha256:"):]
	child := map[string]any{"mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": childDigest, "size": len(files[childPath])}
	forged := map[string]any{"mediaType": child["mediaType"], "digest": childDigest, "size": len(files[childPath]) + 1}
	root := testutil.Descriptor(files, "application/vnd.oci.image.index.v1+json", testutil.JSON(test, map[string]any{"schemaVersion": 2, "manifests": []any{child, forged}}))
	files["index.json"] = testutil.JSON(test, map[string]any{"schemaVersion": 2, "manifests": []any{root}})
	filename := filepath.Join(test.TempDir(), "forged.tar")
	testutil.Tar(test, filename, files)
	if _, err := archive.Inspect(context.Background(), filename, archive.DefaultLimits()); err == nil {
		test.Fatal("forged repeated descriptor accepted")
	}
}

// TestGraphBudget bounds repeated edges independently of unique tar-member count.
func TestGraphBudget(test *testing.T) {
	files, childDigest := testutil.OCI(test, "amd64")
	childPath := "blobs/sha256/" + childDigest[len("sha256:"):]
	child := map[string]any{"mediaType": "application/vnd.oci.image.manifest.v1+json", "digest": childDigest, "size": len(files[childPath])}
	children := make([]any, 100)
	for index := range children {
		children[index] = child
	}
	root := testutil.Descriptor(files, "application/vnd.oci.image.index.v1+json", testutil.JSON(test, map[string]any{"schemaVersion": 2, "manifests": children}))
	files["index.json"] = testutil.JSON(test, map[string]any{"schemaVersion": 2, "manifests": []any{root}})
	filename := filepath.Join(test.TempDir(), "repeated.tar")
	testutil.Tar(test, filename, files)
	limits := archive.DefaultLimits()
	limits.Entries = 20
	if _, err := archive.Inspect(context.Background(), filename, limits); err == nil {
		test.Fatal("reference budget ignored")
	}
}
