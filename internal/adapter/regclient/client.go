// Package regclient adapts the pinned SDK to ImagePort's registry port.
package regclient

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
	sdk "github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/scheme"
	"github.com/regclient/regclient/types/ref"
)

// Secrets is the minimal credential lookup capability needed by the adapter.
type Secrets interface {
	Get(string, bool) (string, error)
}

// Client creates isolated SDK clients so changed credentials cannot remain cached.
type Client struct{ secrets Secrets }

// New constructs an adapter using secrets; no network or Docker config is read.
func New(secrets Secrets) *Client { return &Client{secrets: secrets} }

// sdk creates a TLS-verifying client scoped to one saved registry profile.
func (client *Client) sdk(profile domain.Profile) (*sdk.RegClient, error) {
	return client.sdkProfiles(profile)
}

// sdkProfiles scopes verified TLS and credentials to each participating registry host.
func (client *Client) sdkProfiles(profiles ...domain.Profile) (*sdk.RegClient, error) {
	hosts := make([]config.Host, 0, len(profiles))
	for _, profile := range profiles {
		host, err := client.sdkHost(profile)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, host)
	}
	return sdk.New(sdk.WithConfigHost(hosts...), sdk.WithRetryLimit(2),
		sdk.WithUserAgent("ImagePort"), sdk.WithSlog(slog.New(slog.NewTextHandler(io.Discard, nil)))), nil
}

// sdkHost validates a profile and retrieves credentials without persisting or logging them.
func (client *Client) sdkHost(profile domain.Profile) (config.Host, error) {
	if err := domain.ValidateProfile(profile); err != nil {
		return config.Host{}, err
	}
	password, err := client.secrets.Get(profile.SecretRef, profile.SessionOnly)
	if err != nil {
		return config.Host{}, err
	}
	host := config.HostNewName(profile.Host)
	host.User, host.Pass, host.RegCert = profile.Username, password, profile.CAPEM
	if profile.HTTP {
		host.TLS = config.TLSDisabled
	}
	// SDK diagnostics can contain server-controlled URLs. Only application-level
	// stage/status logs are enabled until a controlled redacting sink is available.
	return *host, nil
}

// Reference validates input and binds short repository paths to the selected host.
func Reference(profile domain.Profile, input string) (ref.Ref, error) {
	input = strings.TrimSpace(input)
	if input == "" || strings.Contains(input, "://") || strings.ContainsAny(input, "\\ \r\n\t") {
		return ref.Ref{}, errors.New("Enter a repository path or full image reference without a URL scheme or spaces")
	}
	if !strings.HasPrefix(input, profile.Host+"/") {
		first, _, full := strings.Cut(input, "/")
		if full && (strings.ContainsAny(first, ".:") || first == "localhost") {
			return ref.Ref{}, errors.New("The image authority does not match this connection; select the correct registry")
		}
		input = profile.Host + "/" + input
	}
	reference, err := ref.New(input)
	if err != nil || reference.Scheme != "reg" || reference.Registry != profile.Host || reference.Repository == "" {
		return ref.Ref{}, errors.New("Invalid image reference; select the matching registry connection")
	}
	return reference, nil
}

// Canonical returns a fully qualified reference; targets must name an explicit tag.
func Canonical(profile domain.Profile, input string, target bool) (string, error) {
	reference, err := Reference(profile, input)
	if err != nil {
		return "", err
	}
	if target && (reference.Digest != "" || !strings.Contains(input[strings.LastIndex(input, "/")+1:], ":")) {
		return "", errors.New("The target must include an explicit tag, for example team/image:v1")
	}
	return reference.CommonName(), nil
}

// Check verifies registry reachability, not repository read/write authorization.
func (client *Client) Check(ctx context.Context, profile domain.Profile) error {
	connection, err := client.sdk(profile)
	if err != nil {
		return err
	}
	reference, err := Reference(profile, "imageport-connectivity")
	if err != nil {
		return err
	}
	_, err = connection.Ping(ctx, reference)
	return registry.Safe(err)
}

// page preserves the server's cursor semantics, including a possible empty last page.
func page(items []string, limit int) domain.Page {
	result := domain.Page{Items: items}
	if len(items) >= limit && len(items) > 0 {
		result.Next = items[len(items)-1]
	}
	return result
}

// Repositories lists one server page; catalog failure leaves direct access available.
func (client *Client) Repositories(ctx context.Context, profile domain.Profile, last string, limit int) (domain.Page, error) {
	connection, err := client.sdk(profile)
	if err != nil {
		return domain.Page{}, err
	}
	listing, err := connection.RepoList(ctx, profile.Host, scheme.WithRepoLast(last), scheme.WithRepoLimit(limit))
	if err != nil {
		return domain.Page{}, registry.Safe(err)
	}
	items, err := listing.GetRepos()
	if err != nil {
		return domain.Page{}, registry.Safe(err)
	}
	return page(items, limit), nil
}

// Tags lists one server page without applying client-side pagination again.
func (client *Client) Tags(ctx context.Context, profile domain.Profile, repository, last string, limit int) (domain.Page, error) {
	connection, err := client.sdk(profile)
	if err != nil {
		return domain.Page{}, err
	}
	reference, err := Reference(profile, repository)
	if err != nil {
		return domain.Page{}, err
	}
	listing, err := connection.TagList(ctx, reference, scheme.WithTagLast(last), scheme.WithTagLimit(limit))
	if err != nil {
		return domain.Page{}, registry.Safe(err)
	}
	items, err := listing.GetTags()
	if err != nil {
		return domain.Page{}, registry.Safe(err)
	}
	return page(items, limit), nil
}

// Digest reads an actual target manifest; missing is distinct from no permission.
func (client *Client) Digest(ctx context.Context, profile domain.Profile, input string) (string, error) {
	connection, err := client.sdk(profile)
	if err != nil {
		return "", err
	}
	reference, err := Reference(profile, input)
	if err != nil {
		return "", err
	}
	image, err := connection.ManifestGet(ctx, reference)
	if err != nil {
		return "", registry.Safe(err)
	}
	return image.GetDescriptor().Digest.String(), nil
}
