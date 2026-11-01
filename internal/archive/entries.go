package archive

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/imneov/ImagePort/internal/domain"
)

type descriptor struct {
	MediaType   string            `json:"mediaType"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	Annotations map[string]string `json:"annotations"`
}
type imageManifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	Manifests     []descriptor `json:"manifests"`
	Config        descriptor   `json:"config"`
	Layers        []descriptor `json:"layers"`
}
type imageConfig struct{ OS, Architecture, Variant string }

// entries detects OCI first; malformed OCI cannot silently fall back to Docker.
func (scanner *scan) entries() (domain.Archive, error) {
	_, layout := scanner.files["oci-layout"]
	_, index := scanner.files["index.json"]
	if layout || index {
		return scanner.ociEntries()
	}
	if _, found := scanner.files["manifest.json"]; found {
		return scanner.dockerEntries()
	}
	return domain.Archive{}, errors.New("Not a docker-save or OCI Layout image archive; container root filesystems and ordinary tar files are not supported")
}

// ociEntries preserves each root descriptor, including full multi-platform indexes.
func (scanner *scan) ociEntries() (domain.Archive, error) {
	var layout struct {
		Version string `json:"imageLayoutVersion"`
	}
	if err := scanner.decode("oci-layout", &layout); err != nil {
		return domain.Archive{}, err
	}
	if layout.Version != "1.0.0" {
		return domain.Archive{}, errors.New("Unsupported OCI Layout version")
	}
	var index imageManifest
	if err := scanner.decode("index.json", &index); err != nil {
		return domain.Archive{}, err
	}
	if index.SchemaVersion != 2 || len(index.Manifests) == 0 {
		return domain.Archive{}, errors.New("The OCI index has no image entries")
	}
	result := domain.Archive{Format: "OCI Layout tar"}
	names := make(map[string]bool)
	for _, entry := range index.Manifests {
		name := entry.Annotations["org.opencontainers.image.ref.name"]
		if name == "" && len(index.Manifests) > 1 {
			return domain.Archive{}, errors.New("Multi-entry OCI archives require a unique reference name for every entry")
		}
		if names[name] {
			return domain.Archive{}, errors.New("The OCI archive has duplicate entry names")
		}
		names[name] = true
		platforms, err := scanner.graph(entry, make(map[string]bool), 0)
		if err != nil {
			return domain.Archive{}, err
		}
		result.Entries = append(result.Entries, domain.ArchiveEntry{Name: name, Digest: entry.Digest, Platforms: platforms})
	}
	return result, nil
}

// graph validates all referenced blobs and recursively reports root platforms.
func (scanner *scan) graph(entry descriptor, visited map[string]bool, depth int) ([]string, error) {
	if err := scanner.ctx.Err(); err != nil {
		return nil, err
	}
	scanner.graphNodes++
	if scanner.graphNodes > scanner.limits.Entries {
		return nil, errors.New("The archive exceeds the reference count safety limit")
	}
	if depth > 16 || visited[entry.Digest] {
		return nil, errors.New("The archive index is cyclic or exceeds the nesting limit")
	}
	visited[entry.Digest] = true
	defer delete(visited, entry.Digest)
	name, err := scanner.blob(entry)
	if err != nil {
		return nil, err
	}
	if platforms, found := scanner.resolved[entry.Digest]; found {
		return platforms, nil
	}
	var manifest imageManifest
	if err := scanner.decode(name, &manifest); err != nil {
		return nil, err
	}
	if manifest.SchemaVersion != 2 {
		return nil, errors.New("Invalid image manifest version")
	}
	platforms, err := scanner.manifestPlatforms(manifest, visited, depth)
	if err != nil {
		return nil, err
	}
	scanner.resolved[entry.Digest] = platforms
	return platforms, nil
}

// manifestPlatforms validates image config/layers or recursively resolves an index.
func (scanner *scan) manifestPlatforms(manifest imageManifest, visited map[string]bool, depth int) ([]string, error) {
	if len(manifest.Manifests) > 0 {
		var platforms []string
		for _, child := range manifest.Manifests {
			children, err := scanner.graph(child, visited, depth+1)
			if err != nil {
				return nil, err
			}
			for _, platform := range children {
				if !slices.Contains(platforms, platform) {
					platforms = append(platforms, platform)
				}
			}
		}
		return platforms, nil
	}
	configName, err := scanner.blob(manifest.Config)
	if err != nil {
		return nil, err
	}
	for _, layer := range manifest.Layers {
		if _, err := scanner.blob(layer); err != nil {
			return nil, err
		}
	}
	platform, err := scanner.platform(configName)
	if err != nil {
		return nil, err
	}
	return []string{platform}, nil
}

// blob verifies descriptor size and returns a checked relative content path.
func (scanner *scan) blob(entry descriptor) (string, error) {
	parts := strings.Split(entry.Digest, ":")
	if len(parts) != 2 {
		return "", errors.New("An archive reference has no valid digest")
	}
	name := "blobs/" + parts[0] + "/" + parts[1]
	file, found := scanner.files[name]
	if !found || file.digest != entry.Digest || file.size != entry.Size {
		return "", errors.New("A referenced archive blob is missing or has an incorrect size or digest")
	}
	return name, nil
}

// decode reads previously bounded JSON and rejects missing/oversized metadata.
func (scanner *scan) decode(name string, target any) error {
	file, exists := scanner.files[name]
	if !exists || len(file.json) == 0 {
		return fmt.Errorf("The archive is missing valid JSON metadata: %s", name)
	}
	if err := json.Unmarshal(file.json, target); err != nil {
		return fmt.Errorf("Invalid archive JSON: %s", name)
	}
	return nil
}

// platform obtains human-readable platform information from an image config.
func (scanner *scan) platform(name string) (string, error) {
	var config imageConfig
	if err := scanner.decode(name, &config); err != nil {
		return "", err
	}
	if config.OS == "" || config.Architecture == "" {
		return "", errors.New("An archive image configuration has no platform information")
	}
	platform := config.OS + "/" + config.Architecture
	if config.Variant != "" {
		platform += "/" + config.Variant
	}
	return platform, nil
}

// dockerEntries requires explicit selection when a docker-save tar has many images.
func (scanner *scan) dockerEntries() (domain.Archive, error) {
	var entries []struct {
		Config           string
		RepoTags, Layers []string
	}
	if err := scanner.decode("manifest.json", &entries); err != nil {
		return domain.Archive{}, err
	}
	if len(entries) == 0 {
		return domain.Archive{}, errors.New("The Docker archive has no image entries")
	}
	result := domain.Archive{Format: "docker save tar"}
	names := make(map[string]bool)
	for _, entry := range entries {
		for _, name := range append([]string{entry.Config}, entry.Layers...) {
			cleaned, err := SafePath(name)
			if err != nil {
				return domain.Archive{}, err
			}
			if _, found := scanner.files[cleaned]; !found {
				return domain.Archive{}, errors.New("The Docker archive is missing a configuration or layer file")
			}
		}
		platform, err := scanner.platform(entry.Config)
		if err != nil {
			return domain.Archive{}, err
		}
		if len(entry.RepoTags) == 0 && len(entries) > 1 {
			return domain.Archive{}, errors.New("A multi-image Docker archive has an untagged entry that cannot be selected safely")
		}
		if len(entry.RepoTags) == 0 {
			entry.RepoTags = []string{""}
		}
		for _, tag := range entry.RepoTags {
			if names[tag] {
				return domain.Archive{}, errors.New("The Docker archive has duplicate entry tags")
			}
			names[tag] = true
			result.Entries = append(result.Entries, domain.ArchiveEntry{Name: tag, Platforms: []string{platform}})
		}
	}
	return result, nil
}
