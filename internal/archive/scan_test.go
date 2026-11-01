package archive_test

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/testutil"
)

// TestInspectFormats validates full OCI graphs and Docker entry/platform detection.
func TestInspectFormats(test *testing.T) {
	oci, expected := testutil.OCI(test, "amd64", "arm64")
	cases := []struct {
		name      string
		files     map[string][]byte
		platforms int
	}{{"oci", oci, 2}, {"docker", testutil.Docker(test), 1}}
	for _, scenario := range cases {
		test.Run(scenario.name, func(test *testing.T) {
			filename := filepath.Join(test.TempDir(), "镜像 archive.tar")
			testutil.Tar(test, filename, scenario.files)
			result, err := archive.Inspect(context.Background(), filename, archive.DefaultLimits())
			if err != nil {
				test.Fatal(err)
			}
			if len(result.Entries) != 1 || len(result.Entries[0].Platforms) != scenario.platforms {
				test.Fatalf("wrong entry: %+v", result)
			}
			if scenario.name == "oci" && result.Entries[0].Digest != expected {
				test.Fatal("root index digest changed")
			}
			data, err := os.ReadFile(filename)
			if err != nil {
				test.Fatal(err)
			}
			checksum := sha256.Sum256(data)
			if result.SHA256 != hex.EncodeToString(checksum[:]) || result.Size != int64(len(data)) {
				test.Fatal("did not hash file through EOF")
			}
		})
	}
}

// TestUnsafeArchive rejects paths, arbitrary tars and tampered/missing OCI blobs.
func TestUnsafeArchive(test *testing.T) {
	for _, scenario := range []string{"traversal", "absolute", "windows", "ordinary", "corrupt", "missing"} {
		test.Run(scenario, func(test *testing.T) {
			files, _ := testutil.OCI(test, "amd64")
			switch scenario {
			case "traversal":
				files["../escape"] = []byte("bad")
			case "absolute":
				files["/escape"] = []byte("bad")
			case "windows":
				files["C:\\escape"] = []byte("bad")
			case "ordinary":
				files = map[string][]byte{"file": []byte("hello")}
			case "corrupt", "missing":
				for name := range files {
					if name != "index.json" && name != "oci-layout" {
						if scenario == "corrupt" {
							files[name] = []byte("tampered")
						} else {
							delete(files, name)
						}
						break
					}
				}
			}
			filename := filepath.Join(test.TempDir(), "bad.tar")
			testutil.Tar(test, filename, files)
			if _, err := archive.Inspect(context.Background(), filename, archive.DefaultLimits()); err == nil {
				test.Fatal("unsafe archive accepted")
			}
		})
	}
}

// TestSpecialAndTruncated rejects links, duplicates and a truncated member body.
func TestSpecialAndTruncated(test *testing.T) {
	for _, scenario := range []string{"link", "duplicate", "truncated"} {
		test.Run(scenario, func(test *testing.T) {
			filename := filepath.Join(test.TempDir(), "bad.tar")
			file, err := os.Create(filename)
			if err != nil {
				test.Fatal(err)
			}
			writer := tar.NewWriter(file)
			header := &tar.Header{Name: "file", Mode: 0600, Size: 8}
			if scenario == "link" {
				header.Typeflag = tar.TypeSymlink
				header.Linkname = "/etc/passwd"
				header.Size = 0
			}
			if err := writer.WriteHeader(header); err != nil {
				test.Fatal(err)
			}
			if scenario != "link" {
				if _, err := writer.Write([]byte("12345678")); err != nil {
					test.Fatal(err)
				}
			}
			if scenario == "duplicate" {
				if err := writer.WriteHeader(header); err != nil {
					test.Fatal(err)
				}
				if _, err := writer.Write([]byte("12345678")); err != nil {
					test.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				test.Fatal(err)
			}
			if err := file.Close(); err != nil {
				test.Fatal(err)
			}
			if scenario == "truncated" {
				if err := os.Truncate(filename, 515); err != nil {
					test.Fatal(err)
				}
			}
			if _, err := archive.Inspect(context.Background(), filename, archive.DefaultLimits()); err == nil {
				test.Fatal("invalid archive accepted")
			}
		})
	}
}

// TestPublishNeverOverwrites covers the race between preflight and final publication.
func TestPublishNeverOverwrites(test *testing.T) {
	directory := test.TempDir()
	partial, final := filepath.Join(directory, "partial"), filepath.Join(directory, "final")
	if err := os.WriteFile(partial, []byte("new"), 0600); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(final, []byte("old"), 0600); err != nil {
		test.Fatal(err)
	}
	if err := archive.Publish(partial, final); err == nil {
		test.Fatal("overwrote target")
	}
	data, err := os.ReadFile(final)
	if err != nil {
		test.Fatal(err)
	}
	if string(data) != "old" {
		test.Fatal("old result changed")
	}
}
