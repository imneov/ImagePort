package regclient

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
)

// DeleteTag uses Harbor's tag endpoint; it never substitutes manifest-wide deletion.
func (client *Client) DeleteTag(ctx context.Context, profile domain.Profile, input, expected string) error {
	reference, err := Reference(profile, input)
	if err != nil {
		return err
	}
	actual, err := client.Digest(ctx, profile, input)
	if err != nil {
		return err
	}
	if actual != expected || expected == "" || reference.Tag == "" {
		return errors.New("The remote version changed after confirmation; refresh and select it again")
	}
	var information struct {
		Version string `json:"harbor_version"`
	}
	if _, err := client.harborRead(ctx, profile, "systeminfo", &information); err != nil {
		return err
	}
	if information.Version == "" {
		return errors.New("This registry does not support deleting a single tag")
	}
	project, repository, valid := strings.Cut(reference.Repository, "/")
	if !valid {
		return errors.New("Select a valid registry project")
	}
	path := "projects/" + url.PathEscape(project) + "/repositories/" + url.PathEscape(url.PathEscape(repository)) + "/artifacts/" + url.PathEscape(expected) + "/tags/" + url.PathEscape(reference.Tag)
	writeErr := client.harborDelete(ctx, profile, path)
	verification, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	actual, err = client.Digest(verification, profile, input)
	if registry.IsCode(err, "NOT_FOUND") {
		return nil
	}
	if err == nil && actual == expected && writeErr != nil {
		return writeErr
	}
	return &registry.Error{Code: "UNKNOWN_RESULT", Message: "The remote tag removal could not be verified; refresh the registry", Action: "Verify interrupted result"}
}

// harborDelete authorizes exactly one product API mutation without redirects or body logging.
func (client *Client) harborDelete(ctx context.Context, profile domain.Profile, path string) error {
	connection, err := client.harborClient(profile)
	if err != nil {
		return err
	}
	defer connection.CloseIdleConnections()
	scheme := "https"
	if profile.HTTP {
		scheme = "http"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, scheme+"://"+profile.Host+"/api/v2.0/"+path, nil)
	if err != nil {
		return registry.Safe(err)
	}
	password, err := client.secrets.Get(profile.SecretRef, profile.SessionOnly)
	if err != nil {
		return err
	}
	if profile.Username != "" {
		request.SetBasicAuth(profile.Username, password)
	}
	response, err := connection.Do(request)
	if err != nil {
		return registry.Safe(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		return harborStatus(response.StatusCode)
	}
	return nil
}
