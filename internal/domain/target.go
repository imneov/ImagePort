package domain

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

// Target is a saved location, independent of the accounts used to access it.
type Target struct {
	ID, Name, Kind, Address string
}

// ValidateTarget validates location metadata without connecting or changing permissions.
func ValidateTarget(target Target) error {
	if strings.TrimSpace(target.Name) == "" || len(target.Name) > 80 {
		return errors.New("Enter a target name of 1–80 characters")
	}
	switch target.Kind {
	case "registry":
		parsed, err := url.Parse(target.Address)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
			return errors.New("Enter an HTTPS registry address without credentials or a repository path")
		}
		profile := Profile{Name: target.Name, Host: parsed.Host, HTTP: parsed.Scheme == "http"}
		if parsed.Scheme != "https" && parsed.Scheme != "http" {
			return errors.New("Enter an HTTPS registry address without credentials or a repository path")
		}
		return ValidateProfile(profile)
	case "filesystem":
		if !filepath.IsAbs(target.Address) {
			return errors.New("Choose an absolute folder path")
		}
	case "engine":
		if strings.ContainsAny(target.Address, "\x00\r\n") {
			return errors.New("Choose an existing Docker context")
		}
	default:
		return errors.New("Choose a registry, local image engine or folder")
	}
	return nil
}
