package main

import "fyne.io/fyne/v2"

// applicationVersion returns the packaged version from metadata, with the source
// development version as a fallback when running an unpackaged application.
func applicationVersion(metadata fyne.AppMetadata) string {
	if releaseVersion := metadata.Custom["releaseVersion"]; releaseVersion != "" {
		return releaseVersion
	}
	if metadata.Release && metadata.Version != "" {
		return metadata.Version
	}
	return version
}
