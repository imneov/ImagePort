// Package docker adapts the official Moby SDK and Docker CLI context store.
package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/url"
	"os"

	"github.com/docker/cli/cli/config"
	contextdocker "github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/moby/moby/client"
)

// Client resolves contexts afresh so an edited endpoint cannot silently change a task.
type Client struct{}

// New constructs the optional Docker adapter without opening a daemon connection.
func New() *Client { return &Client{} }

// selectedContext follows Docker's context/environment precedence without changing it.
func selectedContext() (string, bool, error) {
	if name := os.Getenv("DOCKER_CONTEXT"); name != "" {
		return name, false, nil
	}
	if os.Getenv("DOCKER_HOST") != "" {
		return "default", true, nil
	}
	configuration, err := config.Load(config.Dir())
	if err != nil {
		return "", false, errors.New("Cannot read Docker configuration; check Docker context settings")
	}
	if configuration.CurrentContext != "" {
		return configuration.CurrentContext, false, nil
	}
	return "default", false, nil
}

// contextOptions reads an existing context and its TLS material through Docker's store.
func contextOptions(name string) ([]client.Opt, string, error) {
	configuration := store.NewConfig(nil, store.EndpointTypeGetter("docker", func() any { return &contextdocker.EndpointMeta{} }))
	contexts := store.New(config.ContextStoreDir(), configuration)
	metadata, err := contexts.GetMetadata(name)
	if err != nil {
		return nil, "", errors.New("Cannot read the selected Docker context; check Docker context settings")
	}
	endpoint, err := contextdocker.EndpointFromContext(metadata)
	if err != nil {
		return nil, "", errors.New("The selected Docker context has no Docker endpoint")
	}
	if endpoint.SkipTLSVerify {
		return nil, "", errors.New("The Docker context disables certificate verification; configure a trusted Docker endpoint")
	}
	connection, err := contextdocker.WithTLSData(contexts, name, endpoint)
	if err != nil {
		return nil, "", errors.New("Cannot read Docker context TLS material")
	}
	if err := trustedEndpoint(endpoint.Host, connection.TLSData != nil); err != nil {
		return nil, "", err
	}
	options, err := connection.ClientOpts()
	if err != nil {
		return nil, "", errors.New("Cannot configure the Docker context connection")
	}
	encoded, err := json.Marshal(connection)
	if err != nil {
		return nil, "", errors.New("Cannot identify the Docker context connection")
	}
	return options, fingerprint(encoded), nil
}

// fingerprint identifies an endpoint configuration without retaining certificate keys.
func fingerprint(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

// open binds the SDK to a selected context and checks any previously pinned identity.
func (adapter *Client) open(expected domain.Engine) (*client.Client, domain.Engine, error) {
	name := expected.Context
	fromEnv := expected.FromEnv
	if name == "" {
		selected, environment, err := selectedContext()
		if err != nil {
			return nil, domain.Engine{}, err
		}
		name = selected
		fromEnv = environment
	}
	options := []client.Opt{client.FromEnv}
	if (name == "default" || fromEnv) && os.Getenv("DOCKER_CERT_PATH") != "" && os.Getenv("DOCKER_TLS_VERIFY") == "" {
		return nil, domain.Engine{}, errors.New("Docker TLS certificates are configured without certificate verification; enable DOCKER_TLS_VERIFY")
	}
	identity := fingerprint([]byte(os.Getenv("DOCKER_HOST") + "\x00" + os.Getenv("DOCKER_CERT_PATH") + "\x00" + os.Getenv("DOCKER_TLS_VERIFY")))
	if name != "default" && !fromEnv {
		configured, signature, err := contextOptions(name)
		if err != nil {
			return nil, domain.Engine{}, err
		}
		options, identity = configured, signature
	}
	connection, err := client.New(options...)
	if err != nil {
		return nil, domain.Engine{}, errors.New("Cannot configure the Docker Engine connection")
	}
	if name == "default" || fromEnv {
		if err := trustedEndpoint(connection.DaemonHost(), os.Getenv("DOCKER_CERT_PATH") != "" && os.Getenv("DOCKER_TLS_VERIFY") != ""); err != nil {
			closeClient(connection)
			return nil, domain.Engine{}, err
		}
	}
	engine := domain.Engine{Context: name, Host: connection.DaemonHost(), Identity: identity, FromEnv: fromEnv}
	if expected.Identity != "" && expected != engine {
		closeClient(connection)
		return nil, engine, errors.New("The Docker endpoint changed after selection; refresh local images and create a new operation")
	}
	return connection, engine, nil
}

// trustedEndpoint permits local sockets/loopback and authenticated remote transports.
func trustedEndpoint(host string, verifiedTLS bool) error {
	endpoint, err := url.Parse(host)
	if err != nil {
		return errors.New("Cannot configure the Docker Engine connection")
	}
	if endpoint.User != nil {
		if _, password := endpoint.User.Password(); password {
			return errors.New("Docker endpoint URLs must not contain passwords")
		}
	}
	if endpoint.Scheme == "unix" || endpoint.Scheme == "npipe" || endpoint.Scheme == "fd" || endpoint.Scheme == "ssh" {
		return nil
	}
	if endpoint.Hostname() == "localhost" || net.ParseIP(endpoint.Hostname()).IsLoopback() || verifiedTLS {
		return nil
	}
	return errors.New("Remote Docker TCP endpoints require verified TLS; use a local socket, SSH, or configure Docker TLS certificates")
}

// closeClient reports resource cleanup errors without exposing SDK connection data.
func closeClient(connection *client.Client) {
	if err := connection.Close(); err != nil {
		slog.Warn("Docker client cleanup failed")
	}
}
