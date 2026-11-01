// Package testutil generates tiny deterministic image archives for behavioral tests.
package testutil

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
)

// JSON marshals fixture metadata, failing the owning test on invalid input.
func JSON(test *testing.T, value any) []byte {
	test.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		test.Fatal(err)
	}
	return data
}

// Descriptor returns a content-addressed OCI descriptor and inserts its blob.
func Descriptor(files map[string][]byte, mediaType string, data []byte) map[string]any {
	checksum := digest.FromBytes(data).String()
	files["blobs/"+strings.ReplaceAll(checksum, ":", "/")] = data
	return map[string]any{"mediaType": mediaType, "digest": checksum, "size": len(data)}
}

// OCI creates a runnable multi-platform root or a single-platform manifest root.
func OCI(test *testing.T, platforms ...string) (map[string][]byte, string) {
	test.Helper()
	files := map[string][]byte{"oci-layout": []byte(`{"imageLayoutVersion":"1.0.0"}`)}
	var children []map[string]any
	for _, architecture := range platforms {
		config := JSON(test, map[string]any{"os": "linux", "architecture": architecture, "rootfs": map[string]any{"type": "layers", "diff_ids": []string{}}})
		configuration := Descriptor(files, "application/vnd.oci.image.config.v1+json", config)
		manifest := JSON(test, map[string]any{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.manifest.v1+json", "config": configuration, "layers": []any{}})
		child := Descriptor(files, "application/vnd.oci.image.manifest.v1+json", manifest)
		child["platform"] = map[string]string{"os": "linux", "architecture": architecture}
		children = append(children, child)
	}
	root := children[0]
	if len(children) > 1 {
		index := JSON(test, map[string]any{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.index.v1+json", "manifests": children})
		root = Descriptor(files, "application/vnd.oci.image.index.v1+json", index)
	}
	root["annotations"] = map[string]string{"org.opencontainers.image.ref.name": "fixture"}
	files["index.json"] = JSON(test, map[string]any{"schemaVersion": 2, "manifests": []any{root}})
	return files, root["digest"].(string)
}

// Tar writes sorted regular files to a fixture tar without retaining generated binaries.
func Tar(test *testing.T, filename string, files map[string][]byte) {
	test.Helper()
	file, err := os.Create(filename)
	if err != nil {
		test.Fatal(err)
	}
	writer := tar.NewWriter(file)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(files[name]))}); err != nil {
			test.Fatal(err)
		}
		if _, err := writer.Write(files[name]); err != nil {
			test.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		test.Fatal(err)
	}
	if err := file.Close(); err != nil {
		test.Fatal(err)
	}
}

// Docker generates a real docker-save entry containing an uncompressed layer tar.
func Docker(test *testing.T) map[string][]byte {
	test.Helper()
	var layer bytes.Buffer
	writer := tar.NewWriter(&layer)
	if err := writer.WriteHeader(&tar.Header{Name: "hello.txt", Mode: 0644, Size: 5}); err != nil {
		test.Fatal(err)
	}
	if _, err := writer.Write([]byte("hello")); err != nil {
		test.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		test.Fatal(err)
	}
	configuration := JSON(test, map[string]any{"os": "linux", "architecture": "amd64", "rootfs": map[string]any{"type": "layers", "diff_ids": []string{digest.FromBytes(layer.Bytes()).String()}}})
	name := digest.FromBytes(configuration).Encoded() + ".json"
	manifest := JSON(test, []any{map[string]any{"Config": name, "RepoTags": []string{"fixture:latest"}, "Layers": []string{"layer/layer.tar"}}})
	return map[string][]byte{name: configuration, "layer/layer.tar": layer.Bytes(), "manifest.json": manifest}
}
