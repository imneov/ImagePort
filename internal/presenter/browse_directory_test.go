package presenter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestBrowseDirectoryOffersParentUnlessRoot verifies the file-manager ".." row and its boundary.
func TestBrowseDirectoryOffersParentUnlessRoot(test *testing.T) {
	root := test.TempDir()
	nested := filepath.Join(root, "inner")
	if err := os.MkdirAll(filepath.Join(nested, "leaf"), 0o755); err != nil {
		test.Fatal(err)
	}
	rootPage, err := browseDirectory(context.Background(), root, root)
	if err != nil {
		test.Fatal(err)
	}
	for _, entry := range rootPage.Entries {
		if entry.Kind == "parent" {
			test.Fatal("root must not offer a parent entry")
		}
	}
	nestedPage, err := browseDirectory(context.Background(), root, nested)
	if err != nil {
		test.Fatal(err)
	}
	if len(nestedPage.Entries) == 0 || nestedPage.Entries[0].Kind != "parent" || nestedPage.Entries[0].Name != ".." {
		test.Fatalf("nested folder lacks a leading parent row: %+v", nestedPage.Entries)
	}
	if nestedPage.Entries[0].Path != root {
		test.Fatalf("parent row must point at the actual parent, got %q", nestedPage.Entries[0].Path)
	}
}

// TestParentEntryStaysInsideRoot rejects a parent that would escape the saved folder.
func TestParentEntryStaysInsideRoot(test *testing.T) {
	root := test.TempDir()
	if entry := parentEntry(root, root); entry != nil {
		test.Fatal("root folder must not expose a parent")
	}
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		test.Fatal(err)
	}
	entry := parentEntry(root, child)
	if entry == nil || entry.Path != root {
		test.Fatalf("child folder should return to root: %+v", entry)
	}
}
