package presenter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/distribution/reference"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
)

// BrowseLocation lists one container; callers retain each server cursor without slicing.
func (presenter *Presenter) BrowseLocation(ctx context.Context, target domain.Target, profile domain.Profile, kind, path, cursor string) (domain.LocationPage, error) {
	switch target.Kind {
	case "registry":
		return presenter.browseRegistry(ctx, profile, kind, path, cursor)
	case "engine":
		return presenter.browseEngine(ctx, target.Address)
	case "filesystem":
		return browseDirectory(ctx, target.Address, path)
	default:
		return domain.LocationPage{}, errors.New("Choose a registry, local image engine or folder")
	}
}

// browseRegistry follows product projects when available, otherwise the OCI catalog.
func (presenter *Presenter) browseRegistry(ctx context.Context, profile domain.Profile, kind, path, cursor string) (domain.LocationPage, error) {
	if kind == "reference" {
		return presenter.browseReference(ctx, profile, path)
	}
	var page domain.Page
	var err error
	entryKind := "image"
	product, supported := presenter.Registry.(registry.ProjectBrowser)
	if kind == "root" && supported {
		page, err = product.Projects(ctx, profile, cursor, 50)
		entryKind = "project"
		if registry.IsCode(err, "UNSUPPORTED") {
			page, err = presenter.Registry.Repositories(ctx, profile, cursor, 50)
			entryKind = "image"
		}
	} else if kind == "project" && supported {
		page, err = product.ProjectRepositories(ctx, profile, path, cursor, 50)
	} else if kind == "image" {
		page, err = presenter.Registry.Tags(ctx, profile, path, cursor, 50)
		entryKind = "version"
	} else {
		page, err = presenter.Registry.Repositories(ctx, profile, cursor, 50)
	}
	if err != nil {
		return domain.LocationPage{}, err
	}
	result := domain.LocationPage{Next: page.Next, Total: page.Total, HasTotal: page.HasTotal}
	for _, value := range page.Items {
		entry := domain.LocationEntry{ID: entryKind + ":" + value, Name: value, Kind: entryKind, Path: value}
		if entryKind == "image" && kind == "project" {
			entry.Name = strings.TrimPrefix(value, path+"/")
		}
		if entryKind == "version" {
			entry.ID, entry.Path, entry.Repository = "version:"+path+":"+value, path+":"+value, path
		}
		result.Entries = append(result.Entries, entry)
	}
	return result, nil
}

// browseEngine groups actual daemon tags by image name while preserving every content ID.
func (presenter *Presenter) browseEngine(ctx context.Context, name string) (domain.LocationPage, error) {
	var listing domain.LocalImages
	var err error
	if port, valid := presenter.Local.(interface {
		ImagesIn(context.Context, domain.Engine) (domain.LocalImages, error)
	}); valid {
		listing, err = port.ImagesIn(ctx, domain.Engine{Context: name})
	} else if name == "" {
		listing, err = presenter.Local.Images(ctx)
	} else {
		err = errors.New("The local adapter cannot open this Docker context")
	}
	if err != nil {
		return domain.LocationPage{}, err
	}
	groups := make(map[string][]domain.LocationEntry)
	for _, image := range listing.Images {
		for _, tag := range image.Tags {
			parsed, err := reference.ParseNormalizedNamed(tag)
			if err != nil {
				continue // Engine placeholder tags are not actionable image names.
			}
			tagged, valid := parsed.(reference.Tagged)
			if !valid {
				continue
			}
			name := reference.FamiliarName(parsed)
			groups[name] = append(groups[name], domain.LocationEntry{ID: tag, Kind: "version", Name: tagged.Tag(), Path: tag, Repository: name,
				Size: image.Size, Modified: image.Created, Local: domain.LocalSource{Engine: listing.Engine, ImageID: image.ID}})
		}
		if len(image.Tags) == 0 {
			groups[image.ID] = []domain.LocationEntry{{ID: image.ID, Name: image.ID, Kind: "version", Path: image.ID, Size: image.Size, Local: domain.LocalSource{Engine: listing.Engine, ImageID: image.ID}}}
		}
	}
	result := domain.LocationPage{Engine: listing.Engine}
	for name, children := range groups {
		result.Entries = append(result.Entries, domain.LocationEntry{ID: "image:" + name, Name: name, Kind: "image", Path: name, Children: children})
	}
	sort.Slice(result.Entries, func(left, right int) bool { return result.Entries[left].Name < result.Entries[right].Name })
	return result, nil
}

// browseDirectory lists filesystem items without opening archives or following file symlinks.
func browseDirectory(ctx context.Context, root, path string) (domain.LocationPage, error) {
	if path == "" {
		path = root
	}
	if err := directoryWithin(root, path); err != nil {
		return domain.LocationPage{}, err
	}
	items, err := os.ReadDir(path)
	if err != nil {
		return domain.LocationPage{}, fmt.Errorf("read directory: %w", err)
	}
	result := domain.LocationPage{}
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		info, err := item.Info()
		if err != nil {
			return result, fmt.Errorf("read directory item: %w", err)
		}
		kind := "file"
		if info.IsDir() {
			kind = "folder"
		} else if info.Mode().IsRegular() && strings.HasSuffix(strings.ToLower(item.Name()), ".tar") {
			kind = "archive"
		}
		full := filepath.Join(path, item.Name())
		result.Entries = append(result.Entries, domain.LocationEntry{ID: full, Path: full, Name: item.Name(), Kind: kind, Size: info.Size(), Modified: info.ModTime()})
	}
	sort.SliceStable(result.Entries, func(left, right int) bool {
		return result.Entries[left].Kind == "folder" && result.Entries[right].Kind != "folder"
	})
	return result, nil
}

// directoryWithin checks the resolved directory remains under its saved filesystem root.
func directoryWithin(root, path string) error {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("Choose a folder inside this filesystem target")
	}
	return nil
}
