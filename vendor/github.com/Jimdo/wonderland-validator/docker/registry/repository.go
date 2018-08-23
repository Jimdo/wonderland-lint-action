// Package registry provides access to public and private Docker registries. It can
// be used, for example, to check whether a certain image exists in a registry.
package registry

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/docker/api/types"
	"github.com/genuinetools/reg/registry"

	"golang.org/x/net/context"
)

var DefaultV2Registry = &url.URL{
	Scheme: "https",
	Host:   "registry-1.docker.io",
}

var supportedPublicRegistries = []string{
	"", // Docker Hub
	"quay.io",
}

var supportedPrivateRegistries = []string{
	"registry.jimdo-platform.net",
	"registry.jimdo-platform-stage.net",
}

type Repository struct {
	repository string
	hub        *registry.Registry
	ctx        context.Context
}

func NewRepository(registryName, repoName string, credentialStore auth.CredentialStore) (*Repository, error) {
	if !registryIsSupported(registryName) {
		return nil, fmt.Errorf("registry %s is not supported", registryName)
	}

	endpointURL := getEndpointURL(registryName)
	if !strings.Contains(repoName, "/") && endpointURL == DefaultV2Registry {
		repoName = "library/" + repoName
	}

	username, password := credentialStore.Basic(endpointURL)

	authConfig := types.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: endpointURL.String(),
	}

	hub, err := registry.New(authConfig, registry.Opt{})
	if err != nil {
		return nil, fmt.Errorf("failed to get registry: %s", err)
	}
	ctx := context.Background()
	return &Repository{hub: hub, ctx: ctx, repository: repoName}, nil
}

func registryIsSupported(registryName string) bool {
	registries := append(supportedPublicRegistries, supportedPrivateRegistries...)
	for _, r := range registries {
		if registryName == r {
			return true
		}
	}
	return false
}

func getEndpointURL(registryName string) *url.URL {
	if registryName == "" {
		return DefaultV2Registry
	}
	return &url.URL{
		Scheme: "https",
		Host:   registryName,
	}
}

func (r *Repository) Tags() ([]string, error) {
	return r.hub.Tags(r.repository)
}

func (r *Repository) HasTag(tagName string) (bool, error) {
	tags, err := r.Tags()
	if err != nil {
		if strings.Contains(err.Error(), "repository name not known to registry") {
			return false, nil
		}
		return false, err
	}
	for _, t := range tags {
		if t == tagName {
			return true, nil
		}
	}
	return false, nil
}
