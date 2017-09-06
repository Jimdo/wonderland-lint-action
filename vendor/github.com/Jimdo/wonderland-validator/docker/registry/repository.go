// This package provides access to public and private Docker registries. It can
// be used, for example, to check whether a certain image exists in a registry.
package registry

// The code is based on:
//  - https://github.com/docker/distribution/blob/603ffd58e18a9744679f741f2672dd9aea6babe0/registry/proxy/proxyauth.go
//  - https://github.com/docker/docker/blob/0f41761290160fbce38b6db62916d3009967954c/graph/registry.go

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/auth/challenge"

	"github.com/docker/distribution/registry/client/transport"
	"golang.org/x/net/context"
)

var supportedPublicRegistries = []string{
	"", // Docker Hub
	"quay.io",
}

var supportedPrivateRegistries = []string{
	"registry.jimdo-platform.net",
	"registry.jimdo-platform-stage.net",
}

type Repository struct {
	repo distribution.Repository
	ctx  context.Context
}

func NewRepository(registryName, repoName string, credentialStore auth.CredentialStore) (*Repository, error) {
	if !registryIsSupported(registryName) {
		return nil, fmt.Errorf("registry %s is not supported", registryName)
	}

	var endpointURL string
	if registryName == "" {
		endpointURL = "https://registry-1.docker.io"
		if !strings.Contains(repoName, "/") {
			repoName = "library/" + repoName
		}
	} else {
		endpointURL = "https://" + registryName
	}

	challengeManager := challenge.NewSimpleManager()
	if err := ping(challengeManager, endpointURL+"/v2/"); err != nil {
		return nil, err
	}

	actions := []string{"pull"}
	tokenHandler := auth.NewTokenHandler(
		http.DefaultTransport,
		credentialStore,
		repoName,
		actions...,
	)
	basicHandler := auth.NewBasicHandler(credentialStore)
	authorizer := auth.NewAuthorizer(
		challengeManager,
		tokenHandler,
		basicHandler,
	)
	tr := transport.NewTransport(http.DefaultTransport, authorizer)
	ctx := context.Background()
	repoNameRef, err := reference.WithName(repoName)
	if err != nil {
		return nil, err
	}
	repo, err := client.NewRepository(ctx, repoNameRef, endpointURL, tr)
	if err != nil {
		return nil, err
	}
	return &Repository{repo, ctx}, nil
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

func (r *Repository) Tags() ([]string, error) {
	tagService := r.repo.Tags(r.ctx)
	return tagService.All(r.ctx)
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

func ping(manager challenge.Manager, endpoint string) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("unexpected response from %s (%s)", endpoint, resp.Status)
	}
	defer resp.Body.Close()

	if err := manager.AddResponse(resp); err != nil {
		return err
	}

	return nil
}
