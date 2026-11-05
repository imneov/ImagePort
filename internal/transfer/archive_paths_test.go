package transfer_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/testutil"
)

// TestDockerImportWithNativeTemporaryPath verifies registry upload and export when
// the OS temp directory contains characters outside registry reference syntax.
func TestDockerImportWithNativeTemporaryPath(test *testing.T) {
	directory := filepath.Join(test.TempDir(), "ImagePort 归档@source")
	if err := os.Mkdir(directory, 0700); err != nil {
		test.Fatal(err)
	}
	for _, variable := range []string{"TMPDIR", "TMP", "TEMP"} {
		test.Setenv(variable, directory)
	}
	env := setup(test)
	_, target := seed(test, env, testutil.Docker(test), "native-path")
	sources, err := env.port.Resolve(context.Background(), env.profile, target)
	if err != nil || len(sources) != 1 {
		test.Fatalf("resolve imported image: %+v %v", sources, err)
	}
	path := filepath.Join(env.directory, "source.tar")
	testutil.Tar(test, path, testutil.Docker(test))
	inspection, err := archive.Inspect(context.Background(), path, archive.DefaultLimits())
	if err != nil {
		test.Fatal(err)
	}
	var output bytes.Buffer
	configuration, err := env.port.ExportEntry(context.Background(), path, inspection.Entries[0], "linux/amd64", "test/image:v1", &output)
	if err != nil || configuration == "" || output.Len() == 0 {
		test.Fatalf("export normalized image: config=%s bytes=%d error=%v", configuration, output.Len(), err)
	}
	if sources[0].Platform != "linux/amd64" || sources[0].Digest == "" {
		test.Fatalf("imported image lost its platform or digest: %+v", sources[0])
	}
}
