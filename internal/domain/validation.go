package domain

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"github.com/opencontainers/go-digest"
	"net"
	"net/url"
	"strings"
)

// NewID returns a cryptographically random opaque identifier or an entropy error.
func NewID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

// ValidateLocalSource checks the pinned engine and complete immutable daemon image ID.
func ValidateLocalSource(source LocalSource) error {
	if source.Engine.Identity == "" || source.Engine.Context == "" || source.Engine.Host == "" || !strings.HasPrefix(source.ImageID, "sha256:") {
		return errors.New("Refresh local images and select an immutable image ID")
	}
	if err := digest.Digest(source.ImageID).Validate(); err != nil {
		return errors.New("Refresh local images and select an immutable image ID")
	}
	return nil
}

// ValidateProfile checks metadata and trust without contacting the registry.
func ValidateProfile(profile Profile) error {
	if strings.TrimSpace(profile.Name) == "" {
		return errors.New("Enter a display name for the registry")
	}
	parsed, err := url.Parse("https://" + profile.Host)
	if err != nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil ||
		parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" ||
		strings.ContainsAny(profile.Host, "\\ \t\r\n") {
		return errors.New("Enter only a hostname and optional port, for example registry.example.com")
	}
	if profile.HTTP && parsed.Hostname() != "localhost" && !net.ParseIP(parsed.Hostname()).IsLoopback() {
		return errors.New("HTTP is allowed only for loopback development registries; remote registries require HTTPS")
	}
	if profile.CAPEM != "" && !x509.NewCertPool().AppendCertsFromPEM([]byte(profile.CAPEM)) {
		return errors.New("The CA file does not contain a valid PEM certificate")
	}
	if profile.Username == "" && profile.SecretRef != "" {
		return errors.New("Account authentication requires a username")
	}
	return nil
}
