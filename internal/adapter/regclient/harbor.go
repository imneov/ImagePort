package regclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
)

// harborClient uses profile trust and credentials only for this exact origin.
func (client *Client) harborClient(profile domain.Profile) (*http.Client, error) {
	if err := domain.ValidateProfile(profile); err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if profile.CAPEM != "" {
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, registry.Safe(err)
		}
		if !roots.AppendCertsFromPEM([]byte(profile.CAPEM)) {
			return nil, errors.New("The CA file does not contain a valid PEM certificate")
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	}
	return &http.Client{Transport: transport, Timeout: 25 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, nil
}

// harborRead decodes a bounded JSON response without exposing server diagnostics.
func (client *Client) harborRead(ctx context.Context, profile domain.Profile, path string, output any) (http.Header, error) {
	connection, err := client.harborClient(profile)
	if err != nil {
		return nil, err
	}
	scheme := "https"
	if profile.HTTP {
		scheme = "http"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, scheme+"://"+profile.Host+"/api/v2.0/"+path, nil)
	if err != nil {
		return nil, registry.Safe(err)
	}
	password, err := client.secrets.Get(profile.SecretRef, profile.SessionOnly)
	if err != nil {
		return nil, err
	}
	if profile.Username != "" {
		request.SetBasicAuth(profile.Username, password)
	}
	request.Header.Set("Accept", "application/json")
	response, err := connection.Do(request)
	if err != nil {
		return nil, registry.Safe(err)
	}
	defer response.Body.Close()
	defer connection.CloseIdleConnections()
	if response.StatusCode != http.StatusOK {
		return nil, harborStatus(response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(output); err != nil {
		return nil, registry.Safe(err)
	}
	return response.Header, nil
}

// harborStatus distinguishes unavailable product APIs from account authorization.
func harborStatus(status int) error {
	switch status {
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return &registry.Error{Code: "UNSUPPORTED", Message: "This registry does not expose the Harbor project API", Action: "Use repository browsing instead"}
	case http.StatusUnauthorized:
		return &registry.Error{Code: "UNAUTHORIZED", Message: "Authentication failed or sign-in is required", Action: "Check your username, password, or access token"}
	case http.StatusForbidden:
		return &registry.Error{Code: "FORBIDDEN", Message: "This account is not authorized for this operation", Action: "Ask the registry administrator for the required repository permissions"}
	default:
		return &registry.Error{Code: "REGISTRY", Message: "The project API did not complete", Action: "Check the registry address and network access", Retryable: true}
	}
}

// harborQuery validates page cursors before constructing product API parameters.
func harborQuery(cursor string, limit int) (url.Values, int, error) {
	page := 1
	if cursor != "" {
		value, err := strconv.Atoi(cursor)
		if err != nil || value < 1 || value > 1000000 {
			return nil, 0, errors.New("Invalid registry page cursor")
		}
		page = value
	}
	if limit < 1 || limit > 100 {
		return nil, 0, errors.New("Registry page size must be between 1 and 100")
	}
	return url.Values{"page": {strconv.Itoa(page)}, "page_size": {strconv.Itoa(limit)}, "sort": {"name"}}, page, nil
}

// harborPage preserves the product's total and next server page without slicing.
func harborPage(items []string, header http.Header, number, limit int) domain.Page {
	result := domain.Page{Items: items}
	if total, err := strconv.Atoi(header.Get("X-Total-Count")); err == nil && total >= 0 {
		result.Total, result.HasTotal = total, true
	}
	if (result.HasTotal && number*limit < result.Total) || (!result.HasTotal && len(items) == limit) {
		result.Next = strconv.Itoa(number + 1)
	}
	return result
}

// Projects detects Harbor and lists one page of projects visible to this account.
func (client *Client) Projects(ctx context.Context, profile domain.Profile, cursor string, limit int) (domain.Page, error) {
	query, number, err := harborQuery(cursor, limit)
	if err != nil {
		return domain.Page{}, err
	}
	if cursor == "" {
		var information struct {
			Version string `json:"harbor_version"`
		}
		if _, err := client.harborRead(ctx, profile, "systeminfo", &information); err != nil {
			return domain.Page{}, err
		}
		if information.Version == "" {
			return domain.Page{}, harborStatus(http.StatusNotFound)
		}
	}
	var projects []struct {
		Name string `json:"name"`
	}
	header, err := client.harborRead(ctx, profile, "projects?"+query.Encode(), &projects)
	if err != nil {
		return domain.Page{}, err
	}
	items := make([]string, len(projects))
	for index, project := range projects {
		items[index] = project.Name
	}
	return harborPage(items, header, number, limit), nil
}

// ProjectRepositories lists authorized repositories using Harbor's project API.
func (client *Client) ProjectRepositories(ctx context.Context, profile domain.Profile, project, cursor string, limit int) (domain.Page, error) {
	if project == "" || strings.ContainsAny(project, "/\\\r\n\t ") || project == "." || project == ".." {
		return domain.Page{}, errors.New("Select a valid registry project")
	}
	query, number, err := harborQuery(cursor, limit)
	if err != nil {
		return domain.Page{}, err
	}
	var repositories []struct {
		Name string `json:"name"`
	}
	header, err := client.harborRead(ctx, profile, "projects/"+url.PathEscape(project)+"/repositories?"+query.Encode(), &repositories)
	if err != nil {
		return domain.Page{}, err
	}
	items := make([]string, len(repositories))
	for index, repository := range repositories {
		items[index] = repository.Name
	}
	return harborPage(items, header, number, limit), nil
}
